package types

// PhysicalBodyDescriptor is set as UserData on Box2D Physical bodies to be able to determine collider and collidee from Box2D contact callbacks
type PhysicalBodyDescriptor struct {
	Type _physicaltype
	ID   string
}

type _physicaltype string

func (t _physicaltype) String() string {
	switch t {
	case PhysicalBodyDescriptorType.Obstacle:
		return "Obstacle"
	case PhysicalBodyDescriptorType.Agent:
		return "Agent"
	case PhysicalBodyDescriptorType.Ground:
		return "Ground"
	case PhysicalBodyDescriptorType.Projectile:
		return "Projectile"
	}

	return "UnkownType"
}

var PhysicalBodyDescriptorType = struct {
	Obstacle   _physicaltype
	Agent      _physicaltype
	Ground     _physicaltype
	Projectile _physicaltype
}{
	Obstacle:   _physicaltype("o"),
	Agent:      _physicaltype("a"),
	Ground:     _physicaltype("g"),
	Projectile: _physicaltype("p"),
}

func MakePhysicalBodyDescriptor(type_ _physicaltype, id string) PhysicalBodyDescriptor {
	return PhysicalBodyDescriptor{
		Type: type_,
		ID:   id,
	}
}
