#!/usr/bin/env python
import os
import time
import sys
import datetime

import boto3

from kite.emr.constants import BUNDLE_DIR
from kite.emr.utils import yaml_ordered_load


MEMORY_MB = (2<<30)/(1<<20) # 2GB in MB

JOBURL_TEMPLATE = "https://us-west-1.console.aws.amazon.com/elasticmapreduce/home?region=us-west-1#cluster-details:%s"

SUPPORTED_EXTENSIONS = ['py']

def setup_hadoop_debugging_step():
    return {
        'Name': 'Setup Hadoop Debugging',
        'ActionOnFailure': 'TERMINATE_CLUSTER',
        'HadoopJarStep': {
            'Jar': 's3://us-west-1.elasticmapreduce/libs/script-runner/script-runner.jar',
            'Args': ['s3://us-west-1.elasticmapreduce/libs/state-pusher/0.1/fetch'],
        },
    }

# set env variables http://docs.aws.amazon.com/ElasticMapReduce/latest/ReleaseGuide/emr-release-differences.html#d0e27877
def get_hadoop_env_configs():
    access = os.environ.get("AWS_ACCESS_KEY_ID")
    if access == "":
        raise ValueError("AWS_ACCESS_KEY_ID is not set")

    secret = os.environ.get("AWS_SECRET_ACCESS_KEY")
    if secret == "":
        raise ValueError("AWS_SECRET_ACCESS_KEY is not set")

    return [
        {
            'Classification': 'hadoop-env',
            'Configurations': [
                {
                    'Classification': 'export',
                    'Properties': {
                        'AWS_ACCESS_KEY_ID': access,
                        'AWS_SECRET_ACCESS_KEY': secret,
                    },
                },
            ],
        },
        {
            'Classification': 'hdfs-site',
            'Properties': {
                'dfs.replication': '2',
            },
        },
    ]

class Step(object):
    def __init__(self, name, params, path):
        self.bundledir = BUNDLE_DIR
        self.name = name
        self.params = params
        self.path = path

    def base(self):
        return os.path.join(self.path.path, self.name)

    def s3_base(self):
        return os.path.join(self.path.s3_path(), self.name)

    def output_path(self):
        return os.path.join(self.s3_base(), "output") + os.sep

    def output_exists(self):
        return 'Contents' in boto3.client('s3', region_name='us-west-1').list_objects(
            Bucket=self.path.bucket,
            MaxKeys=1,
            Prefix=os.path.join(self.base(), "output"),
        )

    def _exists(self, name):
        n = os.path.join(self.bundledir, self.name, name)
        return os.path.exists(n)

    def files(self):
        ret = []
        for name in ['mapper', 'reducer']:
            fn = self._find(name)
            if fn != None:
                ret.append(os.path.join(self.s3_base(), fn))
        return ret

    ## --

    def _find(self, name):
        if self._exists(name):
            return name
        for ext in SUPPORTED_EXTENSIONS:
            fn = "%s.%s" % (name, ext)
            if self._exists(fn):
                return fn
        return None

    def mapper(self):
        mapper = self._find('mapper')
        return mapper if mapper else '/bin/cat'

    def reducer(self):
        reducer = self._find('reducer')
        return reducer if reducer else '/bin/cat'

    def _resolve_output(self):
        output = self.params.get('output', '')
        if output == "":
            return self.output_path()
        raise Exception("unknown output spec:", output)

    def _resolve_input(self, steps):
        ret = []
        inputs = self.params.get('input', '')
        for input in inputs.split(','):
            input = input.strip()
            if len(input) == 0:
                continue
            if input in steps:
                ret.append(os.path.join(steps[input].output_path(), '*'))
            elif input.startswith("s3://"):
                ret.append(input)
            else:
                raise Exception("unknown input spec:", input)
        return ret

    # Note that mapredice.{map, reduce}.memory.mb is the size of the container
    # that Hadoop creates to run the map or reduce job. mapreduce.{map,reduce}.java.opts
    # is the max heap size of the JVM that is created to run within that container.
    # This is why *.java.opts must be less than *.memory.mb. We use 64 somewhat
    # arbitrarily here...
    def build_step(self, steps):
        if self.params.has_key("mapreduce_map_memory_mb"):
            map_memory = self.params.get("mapreduce_map_memory_mb")
        else:
            map_memory = self.params.get("mapreduce_memory_mb", MEMORY_MB)
        
        if self.params.has_key("mapreduce_reduce_memory_mb"):
            reduce_memory = self.params.get("mapreduce_reduce_memory_mb")
        else:
            reduce_memory = self.params.get("mapreduce_memory_mb", MEMORY_MB)

        args = [ 'hadoop-streaming',
            "-D", "mapreduce.map.memory.mb=%s" % str(map_memory),
            "-D", "mapreduce.map.java.opts=%s" % (
                   "-Xmx"+str(int(map_memory-64))+"m"),
            "-D", "mapreduce.reduce.memory.mb=%s" % str(reduce_memory),
            "-D", "mapreduce.reduce.java.opts=%s" % (
                   "-Xmx"+str(int(reduce_memory-64))+"m"),
            "-D", "mapreduce.job.jvm.numtasks=1",
            "-D", "dfs.replication=2"]

        if self.params.has_key('mapreduce_reduce_tasks'):
            args.extend(['-D', 'mapreduce.job.reduces=%d' % (
                self.params.get('mapreduce_reduce_tasks'))])

        if self.params.has_key('mapreduce_reduce_running_limit'):
            args.extend(['-D', 'mapreduce.job.running.reduce.limit=%d' % (
                self.params.get('mapreduce_reduce_running_limit'))])
        
        args.extend(['-D', 'mapreduce.map.maxattempts=%d' % (
            self.params.get('mapreduce_map_maxattempts', 1))])

        args.extend(['-D', 'mapreduce.reduce.maxattempts=%d' % (
            self.params.get('mapreduce_reduce_maxattempts', 1))])

        files = self.files()
        if len(files) > 0:
            args.extend(['-files', '%s' % ','.join(files)])

        args.extend([
            '-mapper', '%s' % self.mapper(),
            '-reducer', '%s' % self.reducer(),
            '-input', '%s' % ','.join(self._resolve_input(steps)),
            '-output', '%s' % self._resolve_output(),
        ])

        return {
            'Name': self.name,
            'ActionOnFailure': 'TERMINATE_CLUSTER',
            'HadoopJarStep': {
                'Jar': 'command-runner.jar',
                'Args': args,
            },
        }

class Pipeline(object):
    """
    Pipeline takes in a pipeline.yaml file and a Path object and constructs
    an Elastic Map Reduce jobflow referencing the steps in pipeline.yaml. Each
    step of pipeline.yaml points to a directory of the same name. In each step's
    directory, there should be a 'mapper' and/or 'reducer'. If either isn't set,
    the default of /bin/cat will be used.
    """

    def __init__(self, filename, path):
        with open(filename) as fp:
            self._yaml = yaml_ordered_load(fp)

        self.path = path
        self.config = self._yaml['config']
        self.steps = []
        self.step_map = {}
        self.jobid = None
        self.client = boto3.client('emr', region_name='us-west-1')

        for name, params in self._yaml['pipeline'].iteritems():
            identity = params.get('identity', False)
            if not identity:
                if not os.path.exists(name):
                    raise Exception("could not find directory for step: %s" % name)
                if not os.path.isdir(name):
                    raise Exception("%s is not a directory" % name)

            step = Step(name, params, self.path)
            self.steps.append(step)
            self.step_map[name] = step

    def _get_steps(self):
        steps = [setup_hadoop_debugging_step()]
        for step in self.steps:
            if not step.output_exists():
                steps.append(step.build_step(self.step_map))
        return steps

    def _get_bootstrap_actions(self):
        return [{
            'Name': 'Install kite-python',
            'ScriptBootstrapAction': {
                'Path': os.path.join(self.path.s3_path(), 'bootstrap', 'bootstrap.sh'),
            },
        }]

    def _get_instances(self):        
        n_core_instances = self.config.get('instances', 10)

        env_configs = get_hadoop_env_configs()

        core = {
            'InstanceRole': 'CORE',
            'InstanceType': self.config.get('instance_type', 'm3.xlarge'),
            'InstanceCount': n_core_instances,
            'Configurations': env_configs,
        }
        
        if self.config.has_key('ebs_vol_gb'):
            core['EbsConfiguration'] = {
                'EbsBlockDeviceConfigs': [
                    {
                        'VolumeSpecification': {
                            'VolumeType': self.config.get('ebs_vol_type', 'gp2'),
                            'SizeInGB': self.config.get('ebs_vol_gb'),
                        },
                        'VolumesPerInstance': self.config.get('ebs_vols_per_instance', 1),
                    },
                ],
                'EbsOptimized': False,
            }

        # see http://docs.aws.amazon.com/ElasticMapReduce/latest/ManagementGuide/emr-plan-instances.html
        master_type = 'm3.xlarge'
        if n_core_instances > 49:
            master_type = 'm3.2xlarge'

        return {
            'InstanceGroups': [
                {
                        'InstanceRole': 'MASTER',
                        'InstanceType': master_type,
                        'InstanceCount': 1,
                        'Configurations': env_configs,
                },
                core,
            ],
            'Ec2KeyName': 'kite-dev',
            'Placement': {
                'AvailabilityZone': 'us-west-1b',
            },
            'TerminationProtected': False,
        }

    def start(self):
        self.started_at = datetime.datetime.now()
        resp = self.client.run_job_flow(
            Name=self.path.path,
            LogUri=os.path.join(self.path.s3_path(), 'logs'),
            ReleaseLabel='emr-4.7.0',
            Instances=self._get_instances(),
            Steps=self._get_steps(),
            BootstrapActions=self._get_bootstrap_actions(),
            VisibleToAllUsers=True,
            Configurations=get_hadoop_env_configs(),
            JobFlowRole='EMR_EC2_DefaultRole',
            ServiceRole='EMR_DefaultRole'
        )
        self.jobid = resp['JobFlowId']

    def wait(self):
        """Wait for job to complete."""

        if self.jobid == None:
            raise Exception("job hasn't started yet!")
        
        exit_states = [u'COMPLETED', u'FAILED', u'TERMINATED']

        state, prev = None, None
        while state not in exit_states:
            clusters = self.client.list_clusters(CreatedAfter=self.started_at)
            for c in clusters['Clusters']:
                if c['Id'] == self.jobid:
                    cluster = c
                    break
            else:
                sys.exit('Unable to find cluster {0}'.format(self.jobid))

            state = cluster['Status']['State']
            if state != prev:
                ts = time.strftime('%Y-%m-%d %I:%M:%S %p')
                print("%s %s %s" % (ts, self.jobid, state))
                prev = state
            time.sleep(30)

    def run(self):
        """Start and wait for pipeline."""
        self.start()
        print("view job at: %s" % (JOBURL_TEMPLATE % self.jobid))
        self.wait()

    def describe(self):
        """Describe the pipeline steps."""

        print("base: %s" % self.path.path)
        print("s3 base: %s" % self.path.s3_path())
        print("pipeline steps:")

        for step in self.steps:
            print(step.name)
            if step.output_exists():
                print("\tSKIPPING: output exists: %s" % step.output_path())
            else:
                print("\tmapper: %s" % step.mapper())
                print("\treducer: %s" % step.reducer())
