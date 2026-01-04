# Docker Buildx Bake configuration for Headjack images
#
# Usage:
#   docker buildx bake              # Build all images locally
#   docker buildx bake base         # Build only base image
#   docker buildx bake --push       # Build and push all images
#
# The base image provides all required functionality.

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
  targets = ["base"]
}

target "base" {
  context    = "images/base"
  dockerfile = "Dockerfile"
  tags       = ["${REGISTRY}/${REPOSITORY}:base", "${REGISTRY}/${REPOSITORY}:base-${TAG}"]
  platforms  = ["linux/amd64", "linux/arm64"]
}
