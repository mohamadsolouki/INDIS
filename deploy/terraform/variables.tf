variable "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}

variable "namespace" {
  description = "Kubernetes namespace to deploy INDIS"
  type        = string
  default     = "indis-prod"
}

variable "helm_repo" {
  description = "Helm repository URL"
  type        = string
  default     = "./../helm"
}

variable "chart_version" {
  description = "Version of the INDIS Helm chart"
  type        = string
  default     = "0.1.0"
}
