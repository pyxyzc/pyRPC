# RPC

net/rpc

- the method’s type is exported.
- the method is exported.
- the method has two arguments, both exported (or builtin) types.
- the method’s second argument is a pointer.
- the method has return type error.

`func (t *T) MethodName(argType T1, replyType *T2) error`

