CREATE TABLE monetizable_new_users_{{ds_nodash}}
WITH (
  external_location ='s3://{{params.bucket}}/monetizable/new_users/{{ds_nodash}}',
  format='JSON'
)
AS WITH
  people AS (
    SELECT
      CAST(properties.user_id AS VARCHAR) AS userid,
      BOOL_OR(properties.windows_domain_membership) AS windows_domain_membership,
      ARBITRARY(properties.cio_experiment_trial_end_v1) AS discount
    FROM mixpanel_people
    GROUP BY 1
  ),
  status AS (
    SELECT
      kite_status_normalized.userid,
      MIN(month) AS activation_month,
      ARBITRARY(properties__os) AS os,
      ARBITRARY(maxmind__country_iso_code) AS country_iso_code,
      ARBITRARY(properties__cpu_threads) AS cpu_threads,
      BOOL_OR(properties__git_found) AS git_found,
      BOOL_OR(properties__atom_installed) AS atom_installed,
      BOOL_OR(properties__intellij_installed) AS intellij_installed,
      false AS pycharm_installed,
      BOOL_OR(properties__sublime3_installed) AS sublime3_installed,
      BOOL_OR(properties__vim_installed) AS vim_installed,
      BOOL_OR(properties__vscode_installed) AS vscode_installed,
      BOOL_OR(SUBSTR(properties__intellij_version, 1, 2) NOT IN ('IC', 'PC')) AS intellij_paid,
      BOOL_OR(properties__plan IN ('pro_yearly', 'pro_monthly', 'pro_trial')) AS trial_or_converted,
      BOOL_OR(properties__plan IN ('pro_yearly', 'pro_monthly')) AS converted
    FROM kite_status_normalized
    LEFT OUTER JOIN monetizable_scores ON kite_status_normalized.userid = monetizable_scores.userid
    WHERE
      event = 'kite_status'
      AND (
        year > {{ prev_execution_date.year }}
        OR (year = {{ prev_execution_date.year }} AND month > {{ prev_execution_date.month }})
        OR (year = {{ prev_execution_date.year }} AND month = {{ prev_execution_date.month }} AND day = {{ prev_execution_date.day }})
      )
      AND (
        year < {{ next_execution_date.year }}
        OR (year = {{ next_execution_date.year }} AND month < {{ next_execution_date.month }})
        OR (year = {{ next_execution_date.year }} AND month = {{ next_execution_date.month }} AND day <= {{ next_execution_date.day }})
      )
      AND kite_status_normalized.userid IS NOT NULL
      AND kite_status_normalized.userid != '0'
      AND (monetizable_scores.timestamp IS NULL OR date_diff('day', from_unixtime(monetizable_scores.timestamp, 'utc'), now()) >= 7)
    GROUP BY 1
  )
SELECT
  status.userid,
  activation_month,
  cast(to_unixtime(current_timestamp) as bigint) timestamp,
  COALESCE(os, '{unknown}') AS os,
  COALESCE(country_iso_code, '{unknown}') AS country_iso_code,
  COALESCE(cpu_threads, 0) cpu_threads,
  COALESCE(git_found, FALSE) AS git_found,
  COALESCE(atom_installed, FALSE) AS atom_installed,
  COALESCE(intellij_installed, FALSE) AS intellij_installed,
  COALESCE(pycharm_installed, FALSE) AS pycharm_installed,
  COALESCE(sublime3_installed, FALSE) AS sublime3_installed,
  COALESCE(vim_installed, FALSE) AS vim_installed,
  COALESCE(vscode_installed, FALSE) AS vscode_installed,
  COALESCE(intellij_paid, FALSE) AS intellij_paid,
  COALESCE(windows_domain_membership, FALSE) AS windows_domain_membership,
  COALESCE(discount, 'no discount') AS discount,
  COALESCE(trial_or_converted, FALSE) AS trial_or_converted,
  COALESCE(converted, FALSE) AS converted

FROM status
LEFT JOIN people
  ON status.userid = people.userid
