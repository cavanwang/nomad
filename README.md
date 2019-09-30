This is forked from nomad official github https://github.com/hashicorp/nomad.

Impl vfio gpu plugin to support qemu gpu passthough.

Container can continue to use original nvidia/gpu plugin and
qemu container can use the new vfio gpu plugin.
