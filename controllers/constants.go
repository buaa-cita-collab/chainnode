package controllers

// Operations
const (
	nothingNeeded = "nothingNeeded"
	buildNeeded   = "buildNeeded"
	updateNeeded  = "updateNeeded"
	rebuildNeeded = "rebuildNeeded"
)

// Update Policy
const (
	AutoUpdate = "AutoUpdate"
	NoUpdate   = "NoUpdate"
)

// Keys used in configmap 
// Only a small amout of them is moved here
const (
	nodeKeyKey = "node_key"
	nodeAddressKey = "node_address"
)
