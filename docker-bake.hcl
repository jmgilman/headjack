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

# Set to true to push images to registry
variable "PUSH" {
  default = false
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
  output     = [PUSH ? "type=registry" : "type=docker"]
}

target "systemd" {
  context    = "images/systemd"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:systemd", "${REGISTRY}/${REPOSITORY}:systemd-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
  output     = [PUSH ? "type=registry" : "type=docker"]
  contexts = {
    base = "target:base"
  }
}

target "dind" {
  context    = "images/dind"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:dind", "${REGISTRY}/${REPOSITORY}:dind-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
  output     = [PUSH ? "type=registry" : "type=docker"]
  contexts = {
    systemd = "target:systemd"
  }
}
