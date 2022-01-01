output "instance_groups" {
  value = { for version, prefix in local.version_key_prefixes : version => [for key, mgr in google_compute_region_instance_group_manager.default : mgr.instance_group if length(regexall("^${prefix}", key)) > 0] }
}
