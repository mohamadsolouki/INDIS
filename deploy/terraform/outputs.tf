output "namespace" {
  description = "The namespace INDIS is deployed in"
  value       = kubernetes_namespace.indis.metadata[0].name
}

output "helm_release_status" {
  description = "Status of the Helm release"
  value       = helm_release.indis.status
}
