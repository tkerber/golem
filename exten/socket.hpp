#ifndef GOLEM_SOCKET_H
#define GOLEM_SOCKET_H

#include <jubatus/msgpack/rpc/transport.h>
extern "C" {
#include <gio/gio.h>
}

namespace msgpack {
namespace rpc {

class socket_builder: public msgpack::rpc::builder::base<socket_builder> {
private:
    GSocket *socket;
public:
    socket_builder(GSocket* socket);
    ~socket_builder();

    std::auto_ptr<client_transport> build(
            session_impl* s,
            const address& addr) const;
};

class socket_listener: public listener::base<socket_listener> {
private:
    GSocket *socket;
public:
    socket_listener(GSocket* socket);
    ~socket_listener();

    std::auto_ptr<server_transport> listen(server_impl* svr) const;
};

} // namespace rpc
} // namespace msgpack

#endif /* GOLEM_SOCKET_H */
