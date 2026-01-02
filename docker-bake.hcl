# Docker Buildx Bake configuration for Headjack images
#
# Usage:
#   docker buildx bake              # Build all images locally
#   docker buildx bake base         # Build only base image
#   docker buildx bake --push       # Build and push all images
#
# The images have dependencies: base -> systemd -> dind
# Bake automatically builds dependencies first.

variable "REGISTRY" {
  default = "ghcr.io"
}

variable "REPOSITORY" {
  default = "gilmanlab/headjack"
}

variable "TAG" {
  default = "latest"
}

# Target group to build all images
group "default" {
  targets = ["base", "systemd", "dind"]
}

target "base" {
  context    = "images/base"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:base", "${REGISTRY}/${REPOSITORY}:base-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
}

target "systemd" {
  context    = "images/systemd"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:systemd", "${REGISTRY}/${REPOSITORY}:systemd-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
  contexts = {
    base = "target:base"
  }
}

target "dind" {
  context    = "images/dind"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:dind", "${REGISTRY}/${REPOSITORY}:dind-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
  contexts = {
    systemd = "target:systemd"
  }
}
