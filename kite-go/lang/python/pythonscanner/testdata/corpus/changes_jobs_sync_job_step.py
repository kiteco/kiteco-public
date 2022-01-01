from __future__ import absolute_import, print_function

import os
import requests

from collections import defaultdict
from datetime import datetime
from flask import current_app
from requests.exceptions import ConnectionError, HTTPError, Timeout, SSLError
from sqlalchemy import distinct, or_
from sqlalchemy.exc import IntegrityError
from sqlalchemy.sql import func

from changes.constants import Status, Result
from changes.config import db, statsreporter
from changes.db.utils import try_create
from changes.jobs.sync_artifact import sync_artifact
from changes.models import (
    ItemOption, JobPhase, JobStep, JobPlan, TestCase, ItemStat,
    FileCoverage, FailureReason, SnapshotImage, Artifact, LogSource, Task
)
from changes.queue.task import tracked_task
from changes.db.utils import get_or_create


QUEUED_RETRY_DELAY = 30

INFRA_FAILURE_REASONS = ['malformed_manifest_json', 'missing_manifest_json']


def abort_step(task):
    step = JobStep.query.get(task.kwargs['step_id'])
    step.status = Status.finished
    step.result = Result.aborted
    db.session.add(step)
    db.session.commit()
    current_app.logger.exception('Unrecoverable exception syncing step %s', step.id)


def is_missing_tests(step, jobplan):
    return _expects_tests(jobplan) and _is_in_last_phase(step) and not _has_tests(step)


def _result_from_failure_reasons(step):
    if step.replacement_id is not None:
        return None
    reasons = [r for r, in db.session.query(
        distinct(FailureReason.reason)
    ).filter(
        FailureReason.step_id == step.id,
    ).all()]
    if any(infra_reason in reasons for infra_reason in INFRA_FAILURE_REASONS):
        return Result.infra_failed
    elif reasons:
        return Result.failed
    return None


def _is_snapshot_job(jobplan):
    """
    Args:
        jobplan (JobPlan): The JobPlan to check.
    Returns:
        bool: Whether this plan is for a job that is creating a snapshot.
    """
    is_snapshot = db.session.query(SnapshotImage.query.filter(
        SnapshotImage.job_id == jobplan.job_id
    ).exists()).scalar()
    return bool(is_snapshot)


def _expects_tests(jobplan):
    """Check whether a jobplan expects tests or not.

    Usually this is encoded within the jobplan itself under a snapshot
    of the ItemOptions associated with the plan, but if not we fall
    back to looking at the plan itself.

    Since snapshot builds never return tests, we override this for
    snapshot builds and never expect tests for them (which allows
    building a snapshot for a plan that has buld.expect-tests enabled)
    """
    if _is_snapshot_job(jobplan):
        return False

    if 'snapshot' in jobplan.data:
        options = jobplan.data['snapshot']['options']
    else:
        options = dict(db.session.query(
            ItemOption.name, ItemOption.value,
        ).filter(
            ItemOption.item_id == jobplan.plan.id,
            ItemOption.name == 'build.expect-tests',
        ))

    return options.get('build.expect-tests') == '1'


def _is_in_last_phase(step):
    # TODO(dcramer): there is probably a better way we can be explicit about
    # this?
    jobphase_query = JobPhase.query.filter(
        JobPhase.job_id == step.job_id,
        JobPhase.id != step.phase_id,
        or_(
            JobPhase.date_started > step.phase.date_started,
            JobPhase.date_started == None,  # NOQA
        )
    )

    return not db.session.query(jobphase_query.exists()).scalar()


def _has_tests(step):
    has_tests = db.session.query(TestCase.query.filter(
        TestCase.step_id == step.id,
    ).exists()).scalar()

    return has_tests


def has_test_failures(step):
    return db.session.query(TestCase.query.filter(
        TestCase.step_id == step.id,
        TestCase.result == Result.failed,
    ).exists()).scalar()


# Extra time to allot snapshot builds, as they are expected to take longer.
_SNAPSHOT_TIMEOUT_BONUS_MINUTES = 40


def has_timed_out(step, jobplan, default_timeout):
    """
    Args:
        default_timeout (int): Timeout in minutes to be used when
            no timeout is specified for this build. Required because
            nothing is expected to run forever.
    """
    if step.status != Status.in_progress:
        # HACK: We don't really want to timeout jobsteps that are
        # stuck for infrastructural reasons here, but we don't yet
        # have code to handle other cases well.
        # To be sure that jobsteps can die, we cast a wide net here.
        if step.status == Status.finished:
            return False

    # date_started is preferred if available, but we fall back to
    # date_created (always set) so jobsteps that never start don't get to run forever.
    start_time = step.date_started or step.date_created

    # TODO(dcramer): we make an assumption that there is a single step
    options = jobplan.get_steps()[0].options

    timeout = int(options.get('build.timeout', '0')) or default_timeout

    # timeout is in minutes
    timeout = timeout * 60

    # Snapshots don't time out.
    if _is_snapshot_job(jobplan):
        timeout += 60 * _SNAPSHOT_TIMEOUT_BONUS_MINUTES

    delta = datetime.utcnow() - start_time
    if delta.total_seconds() > timeout:
        return True

    return False


def record_coverage_stats(step):
    coverage_stats = db.session.query(
        func.sum(FileCoverage.lines_covered).label('lines_covered'),
        func.sum(FileCoverage.lines_uncovered).label('lines_uncovered'),
        func.sum(FileCoverage.diff_lines_covered).label('diff_lines_covered'),
        func.sum(FileCoverage.diff_lines_uncovered).label('diff_lines_uncovered'),
    ).filter(
        FileCoverage.step_id == step.id,
    ).group_by(
        FileCoverage.step_id,
    ).first()

    stat_list = (
        'lines_covered', 'lines_uncovered',
        'diff_lines_covered', 'diff_lines_uncovered',
    )
    for stat_name in stat_list:
        try_create(ItemStat, where={
            'item_id': step.id,
            'name': stat_name,
            'value': getattr(coverage_stats, stat_name, 0) or 0,
        })


# In minutes, the timeout applied to jobs without a timeout specified at build time.
# If the job legitimately takes more than an hour, the build
# should specify an appropriate timeout.
DEFAULT_TIMEOUT_MIN = 60


# In seconds, the timeout applied to any requests we make to the artifacts
# store. Arbitrarily chose as the amount of delay we can tolerate for each
# sync_job_step.
ARTIFACTS_REQUEST_TIMEOUT_SECS = 5


# List of artifact names recognized to be log source (content which is
# continuously updated during the duration of a test, like console logs and
# infralogs)
LOGSOURCE_WHITELIST = ('console', 'infralog',)


def _sync_from_artifact_store(jobstep):
    """Checks and creates new artifacts from the artifact store."""
    url = '{base}/buckets/{jobstep_id}/artifacts/'.format(
        base=current_app.config.get('ARTIFACTS_SERVER'),
        jobstep_id=jobstep.id.hex,
    )
    job = jobstep.job

    try:
        res = requests.get(url, timeout=ARTIFACTS_REQUEST_TIMEOUT_SECS)
        res.raise_for_status()
        artifacts = res.json()
        for artifact in artifacts:
            # Artifact name is guaranteed to be unique in an artifact store bucket.
            artifact_name = artifact['name']
            artifact_path = artifact['relativePath']
            if artifact_name in LOGSOURCE_WHITELIST:
                get_or_create(LogSource, where={
                    'name': artifact_name,
                    'job': job,
                    'step': jobstep,
                }, defaults={
                    'project': job.project,
                    'date_created': job.date_started,
                    'in_artifact_store': True,
                })

                # If this artifact is a logsource, don't add it to the list of
                # test artifacts.
                continue

            art, created = get_or_create(Artifact, where={
                # Don't conflict with same artifacts uploaded by other means (Jenkins/Mesos)
                'name': 'artifactstore/' + artifact_path,
                'step_id': jobstep.id,
                'job_id': jobstep.job_id,
                'project_id': jobstep.project_id,
            })
            if created:
                art.file.storage = 'changes.storage.artifactstore.ArtifactStoreFileStorage'
                filename = 'buckets/{jobstep_id}/artifacts/{artifact_name}'.format(
                    jobstep_id=jobstep.id.hex,
                    artifact_name=artifact_name,
                )
                art.file.save(None, filename)
                try:
                    db.session.add(art)
                    db.session.commit()
                except IntegrityError, err:
                    db.session.rollback()
                    current_app.logger.error(
                        'DB Error while inserting/updating artifact %s: %s', filename, err)
    except (ConnectionError, HTTPError, SSLError, Timeout) as err:
        if isinstance(err, HTTPError) and err.response is not None and err.response.status_code == 404:
            # While not all plans use the Artifact Store, 404s are normal and expected.
            # No sense in reporting them.
            pass
        else:
            # Log to sentry - unable to contact artifacts store
            current_app.logger.warning('Error fetching url %s: %s', url, err, exc_info=True)
    except Exception, err:
        current_app.logger.error('Error updating artifacts for jobstep %s: %s', jobstep, err, exc_info=True)
        raise err


def _get_artifacts_to_sync(artifacts, prefer_artifactstore):
    def is_artifact_store(artifact):
        return artifact.file.storage == 'changes.storage.artifactstore.ArtifactStoreFileStorage'

    artifacts_by_name = defaultdict(list)
    # group by filename
    for artifact in artifacts:
        artifacts_by_name[os.path.basename(artifact.name)].append(artifact)

    to_sync = []
    for name, arts in artifacts_by_name.iteritems():
        artifactstore_arts = [a for a in arts if is_artifact_store(a)]
        other_arts = [a for a in arts if not is_artifact_store(a)]

        # if we have this artifact from both sources, let buildstep choose which to use
        if len(artifactstore_arts) and len(other_arts):
            arts = artifactstore_arts if prefer_artifactstore else other_arts

        to_sync.extend(arts)

    return to_sync


def _sync_artifacts_for_jobstep(step):
    # only generate the sync_artifact tasks for this step once
    if Task.query.filter(
        Task.parent_id == step.id,
        Task.task_name == 'sync_artifact',
    ).first():
        return

    _, buildstep = JobPlan.get_build_step_for_job(job_id=step.job_id)
    prefer_artifactstore = buildstep.prefer_artifactstore()
    artifacts = Artifact.query.filter(Artifact.step_id == step.id).all()

    to_sync = _get_artifacts_to_sync(artifacts, prefer_artifactstore)

    # buildstep may want to check for e.g. required artifacts
    buildstep.verify_final_artifacts(step, to_sync)

    for artifact in to_sync:
        sync_artifact.delay_if_needed(
            artifact_id=artifact.id.hex,
            task_id=artifact.id.hex,
            parent_task_id=step.id.hex,
        )


@tracked_task(on_abort=abort_step, max_retries=100)
def sync_job_step(step_id):
    """
    Polls a jenkins build for updates. May have sync_artifact children.
    """
    step = JobStep.query.get(step_id)
    if not step:
        return

    jobplan, implementation = JobPlan.get_build_step_for_job(job_id=step.job_id)

    # only synchronize if upstream hasn't suggested we're finished
    if step.status != Status.finished:
        implementation.update_step(step=step)

    db.session.flush()

    _sync_from_artifact_store(step)

    if step.status == Status.finished:
        _sync_artifacts_for_jobstep(step)

    is_finished = (step.status == Status.finished and
                   # make sure all child tasks (like sync_artifact) have also finished
                   sync_job_step.verify_all_children() == Status.finished)

    if not is_finished:
        default_timeout = current_app.config['DEFAULT_JOB_TIMEOUT_MIN']
        if has_timed_out(step, jobplan, default_timeout=default_timeout):
            old_status = step.status
            step.data['timed_out'] = True
            implementation.cancel_step(step=step)

            # Not all implementations can actually cancel, but it's dead to us as of now
            # so we mark it as finished.
            step.status = Status.finished
            step.date_finished = datetime.utcnow()

            # Implementations default to marking canceled steps as aborted,
            # but we're not canceling on good terms (it should be done by now)
            # so we consider it a failure here.
            #
            # We check whether the step was marked as in_progress to make a best
            # guess as to whether this is an infrastructure failure, or the
            # repository under test is just taking too long. This won't be 100%
            # reliable, but is probably good enough.
            if old_status == Status.in_progress:
                step.result = Result.failed
            else:
                step.result = Result.infra_failed
            db.session.add(step)

            job = step.job
            try_create(FailureReason, {
                'step_id': step.id,
                'job_id': job.id,
                'build_id': job.build_id,
                'project_id': job.project_id,
                'reason': 'timeout'
            })

            db.session.flush()
            statsreporter.stats().incr('job_step_timed_out')
            # If we timeout something that isn't in progress, that's our fault, and we should know.
            if old_status != Status.in_progress:
                current_app.logger.warning(
                    "Timed out jobstep that wasn't in progress: %s (was %s)", step.id, old_status)

        if step.status != Status.in_progress:
            retry_after = QUEUED_RETRY_DELAY
        else:
            retry_after = None
        raise sync_job_step.NotFinished(retry_after=retry_after)

    # Ignore any 'failures' if the build did not finish properly.
    # NOTE(josiah): we might want to include "unknown" and "skipped" here as
    # well, or have some named condition like "not meaningful_result(step.result)".
    if step.result in (Result.aborted, Result.infra_failed):
        _report_jobstep_result(step)
        return

    # Check for FailureReason objects generated by child jobs
    failure_result = _result_from_failure_reasons(step)
    if failure_result and failure_result != step.result:
        step.result = failure_result
        db.session.add(step)
        db.session.commit()
        if failure_result == Result.infra_failed:
            _report_jobstep_result(step)
            return

    try:
        record_coverage_stats(step)
    except Exception:
        current_app.logger.exception('Failing recording coverage stats for step %s', step.id)

    # We need the start time of this step's phase to determine if we're part of
    # the last phase. So, if date_started is empty, wait for sync_phase to catch
    # up and try again.
    if _expects_tests(jobplan) and not step.phase.date_started:
        current_app.logger.warning(
            "Phase[%s].date_started is missing. Retrying Step", step.phase.id)

        # Reset result to unknown to reduce window where test might be incorrectly green.
        # Set status to in_progress so that the next sync_job_step will fetch status from Jenkins again.
        step.result = Result.unknown
        step.status = Status.in_progress
        raise sync_job_step.NotFinished(retry_after=QUEUED_RETRY_DELAY)

    missing_tests = is_missing_tests(step, jobplan)

    try_create(ItemStat, where={
        'item_id': step.id,
        'name': 'tests_missing',
        'value': int(missing_tests),
    })

    if missing_tests:
        if step.result != Result.failed:
            step.result = Result.failed
            db.session.add(step)

        try_create(FailureReason, {
            'step_id': step.id,
            'job_id': step.job_id,
            'build_id': step.job.build_id,
            'project_id': step.project_id,
            'reason': 'missing_tests'
        })
        db.session.commit()

    db.session.flush()

    if has_test_failures(step):
        if step.result != Result.failed:
            step.result = Result.failed
            db.session.add(step)

        try_create(FailureReason, {
            'step_id': step.id,
            'job_id': step.job_id,
            'build_id': step.job.build_id,
            'project_id': step.project_id,
            'reason': 'test_failures'
        })
        db.session.commit()
    _report_jobstep_result(step)


def _report_jobstep_result(step):
    """To be called once we're done syncing a JobStep to report the result for monitoring.

    Args:
        step (JobStep): The JobStep to report the result of.
    """
    labels = {
        Result.unknown: 'unknown',
        Result.passed: 'passed',
        Result.failed: 'failed',
        Result.infra_failed: 'infra_failed',
        Result.aborted: 'aborted',
        Result.skipped: 'skipped',
    }
    label = labels.get(step.result, 'OTHER')
    # TODO(kylec): Include the project slug in the metric so we can
    # track on a per-project basis if needed.
    statsreporter.stats().incr('jobstep_result_' + label)
