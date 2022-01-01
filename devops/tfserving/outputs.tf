output "instance_ip" {
  value = google_compute_instance.tf_serving_instance.network_interface[0].network_ip
}
