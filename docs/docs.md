# Day1-Server

## Request

server 接受客户端到来的请求，如下所示：

```
err = client.Call("Arith.Multiply", args, &reply)
```

包括：

1. 服务名
2. 方法名
3. 入参
4. 出参（错误+返回值）

参数和返回值抽象为Body，其他抽象为 **Header**

```
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}
```

## Codec

抽象出编解码接口Codec，可以实现不同的CodeC实例

实现类需要实现 ReadHeader / ReadBody / Write，使用`var _ Codec = (*GobCodec)(nil)` 强制检查

Write实现trick

```
func NewGobCodec(conn io.ReadWriteCloser) Codec {
	buf := bufio.NewWriter(conn)
	return &GobCodec{
		conn: conn,
		buf:  buf,
		dec:  gob.NewDecoder(conn),
		enc:  gob.NewEncoder(buf),
	}
}
```

```
func (c *JsonCodec) Write(h *Header, body interface{}) (err error) {
	defer func() {
		_ = c.buf.Flush()
		if err != nil {
			_ = c.Close()
		}
	}()
	if err = c.enc.Encode(h); err != nil {
		log.Println("rpc: json error encoding header:", err)
		return
	}
	if err = c.enc.Encode(body); err != nil {
		log.Println("rpc: json error encoding body:", err)
		return
	}
	return
}
```

使用 buffer 来优化写入效率，先写入到buffer里，再调用 buffer.Flush() 来将 buffer 中的全部内容写入到 conn 中，从而优化效率

## Communication

一次连接的开始，确定「协商」内容，目前只有编解码方式需要确定，后续的请求共享这些 Option

```
| Option | Header1 | Body1 | Header2 | Body2 | ...
```

## Server

经典的通信过程，for{} 循环等待socket建立，开子协程（`go server.ServeConn(conn)`）处理连接

> trick：
>
> 可以启用一个default server，方便用户调用
>
> ```
> var DefaultServer = NewServer()
> func Accept(lis net.Listener) { DefaultServer.Accept(lis) }
> ```

具体的连接处理：`server.ServeConn(conn)`

readRequest -> handleRequest -> sendResponse

回复请求的报文必须是逐个发送，防止交织，一把大锁保平安







# Day2

```
// Call represents an active RPC.
// func (t *T) MethodName(argType T1, replyType *T2) error
type Call struct {
	Seq           uint64
	ServiceMethod string      // format "<service>.<method>"
	Args          interface{} // arguments to the function
	Reply         interface{} // reply from the function
	Error         error       // if error occurs, it will be set
	Done          chan *Call  // Strobes when call is complete.
}
```

```
// Client represents an RPC Client.
// There may be multiple outstanding Calls associated
// with a single Client, and a Client may be used by
// multiple goroutines simultaneously.
type Client struct {
	cc       codec.Codec      // encoder/decoder
	opt      *server.Option   // options
	sending  sync.Mutex       // protect following
	header   codec.Header     // request header
	mu       sync.Mutex       // protect following
	seq      uint64           // request sequence number
	pending  map[uint64]*Call // seq:call
	closing  bool             // user has called Close
	shutdown bool             // server has told us to stop
}
```

为什么Call与Header有公共字段，而不直接在Call中用Header呢？

```
type Header struct {
	ServiceMethod string // format "Service.Method"
	Seq           uint64 // sequence number chosen by client
	Error         string
}
```

或者说

```
seq, err := client.registerCall(call)
    if err != nil {
    call.Error = err
    call.done()
    return
}
// prepare request header
client.header.ServiceMethod = call.ServiceMethod
client.header.Seq = seq
client.header.Error = ""
```

不重用header是语义原因，因为 Header 的存在是为了通讯，而Call身不涉及通讯，其只代表客户端发起的调用，客户端理论上并不知道底层如何调用

整个client：

1. go startServer(addr)启动服务器，服务器开始监听端口
2. pyrpc.Dial("tcp", <-addr)，客户端在对应端口上创建服务器，同时创建一个对应的客户端来进行调用和接受返回
3. Dial会新建一个客户端，并且将options传给服务端，随后根据options的配置新建一个对应的Codec，同时客户端开始调用receive接收信息
4. for循环中client调用call发送信息，call通过go组装好参数后，再由send发送信息，send的过程中客户端通过write将信息发送到服务端
5. 服务端在serveCodec中处理并返回信息后，客户端的receive协程将会收到信息，此时将会读取到回复，对应的call.Done接收到变量，意味着执行结束
6. 执行结束后reply传给了call.Reply，绑定到主函数中的reply string，打印出来即为结果。

# Day3

旧的main.go

```
func startServer(addr chan string) {
	// pick a free port
	l, err := net.Listen("tcp", ":8082")
	if err != nil {
		log.Fatal("network error:", err)
	}
	log.Println("start rpc server on", l.Addr())
	addr <- l.Addr().String()
	server.Accept(l)
}

func main() {
	log.SetFlags(0)
	addr := make(chan string)
	go startServer(addr)

	client, _ := client.Dial("tcp", <-addr)
	defer func() { _ = client.Close() }()

	time.Sleep(time.Second)
	// send request & receive response
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			args := fmt.Sprintf("pyrpc req %d", i)
			var reply string
			if err := client.Call("Foo.Sum", args, &reply); err != nil {
				log.Fatal("call Foo.Sum error:", err)
			}
			log.Println("reply:", reply)
		}(i)
	}
	wg.Wait()
}

```







Day4-Timeout







