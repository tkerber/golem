#include "socket.hpp"
// This line is a bit tricky, as base.h is not part of the official API, but
// is required to extend its functionality. We just add the jubatus-msgpack
// source to the include path.
#include <jubatus/msgpack/rpc/transport/base.h>
#include <jubatus/msgpack/rpc/transport.h>
extern "C" {
#include <gio/gio.h>
}

namespace msgpack {
namespace rpc {

class client_socket : public transport::stream_handler<client_socket> {
public:

    client_socket(int fd, session_impl* s):
            stream_handler<client_socket>(fd, s->get_loop_ref()) {
        this->session = s->shared_from_this();
    }

    ~client_socket() { }

    void on_response(
            msgid_t msgid,
            msgpack::object result,
            msgpack::object error,
            auto_zone z) {
        shared_session s = this->session.lock();
        if(!s) {
            throw transport::closed_exception();
        }
        s->on_response(msgid, result, error, z);
    }

private:
    weak_session session;

};

class socket_transport: public client_transport {
    session_impl *session;
    mp::shared_ptr<client_socket> socket;
    GSocket *g_socket;
public:

    socket_transport(session_impl* s, GSocket *socket) {
        int fd = g_socket_get_fd(socket);
        this->socket = s->get_loop_ref()->add_handler<client_socket>(fd, s);
        this->session = s;
        this->g_socket = socket;
        g_object_ref(socket);
    }

    ~socket_transport() {
        close_sock();
        g_object_unref(g_socket);
    }

public:

    void send_data(msgpack::sbuffer* sbuf) {
        this->socket->send_data(sbuf);
    }

    void send_data(auto_vreflife vbuf) {
        this->socket->send_data(vbuf);
    }
    
    void close() {
        close_sock();
    }

private:

    void close_sock() {
        this->socket->remove_handler();
        this->socket.reset();
    }

};

socket_builder::socket_builder(GSocket *socket) {
    this->socket = socket;
    g_object_ref(this->socket);
}

socket_builder::~socket_builder() {
    g_object_unref(this->socket);
}

std::auto_ptr<client_transport>
socket_builder::build(
        session_impl* s,
        const address& addr) const {
    return std::auto_ptr<client_transport>(
            new socket_transport(s, this->socket));
}

} // namespace rpc
} // namespace msgpack
