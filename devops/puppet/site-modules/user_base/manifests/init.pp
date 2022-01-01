class user_base (
  String $process_name,
) {
  require kite_base

  include user_base::executable
  include user_base::fluentd
}
