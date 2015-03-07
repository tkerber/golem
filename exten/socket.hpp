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

    std::auto_ptr<msgpack::rpc::client_transport> build(
            msgpack::rpc::session_impl* s,
            const msgpack::rpc::address& addr) const;
};

} // namespace rpc
} // namespace msgpack

#endif /* GOLEM_SOCKET_H */
