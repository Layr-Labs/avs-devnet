# Import remote code from another package using an absolute import
ethereum_package = import_module("github.com/ethpandaops/ethereum-package/main.star")

def run(plan, args={}):
    ethereum_package.run(plan, args)
