package sport

type FaultyMode uint64

const (
	// Disabled disables the faulty mode
	Disabled FaultyMode = iota
	// Random attacks randomly
	Random
	// NotBroadcast doesn't broadcast any messages to other validators
	NotBroadcast
	// SendWrongMsg sends the message with the wrong message code
	SendWrongMsg
	// ModifySig modifies the message signature
	ModifySig
	// AlwaysPropose always proposes a proposal to validators
	AlwaysPropose
	// AlwaysRoundChange always sends round change while receiving messages
	AlwaysRoundChange
	// BadBlock always proposes a block with bad body
	BadBlock
)


func (f FaultyMode) Uint64() uint64 {
	return uint64(f)
}

func (f FaultyMode) String() string {
	switch f {
	case Disabled:
		return "Disabled"
	case Random:
		return "Random"
	case NotBroadcast:
		return "NotBroadcast"
	case SendWrongMsg:
		return "SendWrongMsg"
	case ModifySig:
		return "ModifySig"
	case AlwaysPropose:
		return "AlwaysPropose"
	case AlwaysRoundChange:
		return "AlwaysRoundChange"
	case BadBlock:
		return "BadBlock"
	default:
		return "Undefined"
	}
}
