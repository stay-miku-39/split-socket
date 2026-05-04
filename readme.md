
架构：
- split-socket层
    1. 负责上下行分离
    2. 上行对应一个Writer接口，下行对应一个Reader接口
    3. 类似tcp socket的用法,实现Listener接口、Transport接口和Conn接口
    4. 底层通过其他协议实现
    5. 可靠性靠底层传输层
- AES加密/混淆层(可选)
    1. 实现Listener接口、Transport接口和Conn接口
- HTTP层(可选)
    1. 实现Listener接口、Transport接口和Conn接口
    2. 分别实现上行和下行逻辑
    3. http/1.1 http/2
    4. keep-alive连接复用
    5. session机制维护逻辑连接
    6. Client2Server使用超时机制确定连接是否存在，使用心跳帧机制(http1.1)
    7. Server2Client使用断连后超时确定连接是否存在(允许在断开后重连)

- TLS层(可选)
    1. 裸TLS层或TLS+HTTP
    2. tls1.3
- TCP/UDP层(可选)
