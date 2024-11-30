package main

type Capability struct {
	Name      string
	Version   string
	Enabled   bool
	TaskTypes []string
}

type CapabilityRegistry struct {
	capabilities map[string]Capability
}

func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{
		capabilities: make(map[string]Capability),
	}
}

func (r *CapabilityRegistry) Register(cap Capability) {
	r.capabilities[cap.Name] = cap
}
