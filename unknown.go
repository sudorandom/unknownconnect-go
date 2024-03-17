package unknownconnect

import "google.golang.org/protobuf/reflect/protoreflect"

// DropUnknownFields recursively drops any unknown fields from the provided protobuf message.
func DropUnknownFields(msg protoreflect.Message) {
	ForEachUnknownField(msg, func(msg protoreflect.Message) bool {
		msg.SetUnknown(nil)
		return true
	})
}

// MessageHasUnknownFields returns true if the given protoreflect.Message has any unknown fields.
func MessageHasUnknownFields(msg protoreflect.Message) bool {
	var hasUnknown bool
	ForEachUnknownField(msg, func(_ protoreflect.Message) bool {
		hasUnknown = true
		return false
	})
	return hasUnknown
}

// ForEachUnknownField recursively scans the given protoreflect.Message object for unknown fields and calls the given callback
// function when it finds a message containing an unknown field.
func ForEachUnknownField(msg protoreflect.Message, cb func(msg protoreflect.Message) bool) {
	forEachUnknownField(msg, cb)
}

func forEachUnknownField(msg protoreflect.Message, cb func(msg protoreflect.Message) bool) bool {
	if len(msg.GetUnknown()) > 0 {
		if !cb(msg) {
			return false
		}
	}

	doContinue := true
	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		doContinue = forEachFieldUnknownField(fd, v, cb)
		return doContinue
	})
	return doContinue
}

func forEachFieldUnknownField(fd protoreflect.FieldDescriptor, v protoreflect.Value, cb func(msg protoreflect.Message) bool) bool {
	if fd.IsMap() {
		v.Map().Range(func(mk protoreflect.MapKey, mv protoreflect.Value) bool {
			return !forEachFieldUnknownField(fd.MapValue(), mv, cb)
		})
		return true
	}

	switch fd.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		if fd.IsList() {
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				vv := list.Get(i)
				if !forEachUnknownField(vv.Message(), cb) {
					return false
				}
			}
		} else {
			if !forEachUnknownField(v.Message(), cb) {
				return false
			}
		}
	default:
	}
	return true
}
