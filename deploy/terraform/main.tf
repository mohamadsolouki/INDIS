terraform {
  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.23.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11.0"
    }
  }
}

provider "kubernetes" {
  config_path = var.kubeconfig_path
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

resource "kubernetes_namespace" "indis" {
  metadata {
    name = var.namespace
  }
}

resource "helm_release" "indis" {
  name       = "indis"
  repository = var.helm_repo
  chart      = "indis"
  namespace  = kubernetes_namespace.indis.metadata[0].name
  version    = var.chart_version

  values = [
    file("${path.module}/../helm/indis/values-prod.yaml")
  ]
}

resource "kubernetes_storage_class" "baremetal_storage" {
  count = var.enable_baremetal_storage ? 1 : 0
  metadata {
    name = "indis-baremetal-storage"
  }
  storage_provisioner = "kubernetes.io/no-provisioner"
  volume_binding_mode = "WaitForFirstConsumer"
}

resource "kubernetes_network_policy" "default_deny" {
  metadata {
    name      = "default-deny"
    namespace = kubernetes_namespace.indis.metadata[0].name
  }

  spec {
    pod_selector {}
    policy_types = ["Ingress"]
  }
}
