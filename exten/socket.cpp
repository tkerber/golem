#include "socket.hpp"
// This line is a bit tricky, as base.h is not part of the official API, but
// is required to extend its functionality. We just add the jubatus-msgpack
// source to the include path.
#include <jubatus/msgpack/rpc/transport.h>
#include <jubatus/msgpack/rpc/transport_impl.h>
#include <jubatus/msgpack/rpc/server_impl.h>
#include <jubatus/msgpack/rpc/session_impl.h>
#include <jubatus/msgpack/rpc/protocol.h>
#include <jubatus/msgpack/rpc/types.h>
extern "C" {
#include <gio/gio.h>
}

#ifndef MSGPACK_RPC_STREAM_BUFFER_SIZE
#define MSGPACK_RPC_STREAM_BUFFER_SIZE (256*1024)
#endif

#ifndef MSGPACK_RPC_STREAM_RESERVE_SIZE
#define MSGPACK_RPC_STREAM_RESERVE_SIZE (32*1024)
#endif

namespace msgpack {
namespace rpc {

namespace transport {

// BEGIN code copied from jubatus/msgpack/rpc/transport/base.h.
// This code was copied almost 1:1, but couldn't be imported
// directly due to problems with cclog.
//
// This code may be found at github.com/jubatus/jubatus-msgpack-rpc
// under the apache software license.
//
// The rest of the code in this file is closely modelled on the code
// in jubatus/msgpack/rpc/transport/.
//
// The following copyrights apply:
// Copyright (C) 2009-2010 FURUHASHI Sadayuki
// Copyright (C) 2013 Preferred Infrastructure and Nippon Telegraph and Telephone Corporation.

class closed_exception { };

template <typename MixIn>
class stream_handler : public mp::wavy::handler, public message_sendable {
public:
	stream_handler(int fd, loop lo);
	~stream_handler();

	void remove_handler();

	mp::shared_ptr<message_sendable> get_response_sender();

	// message_sendable interface
	void send_data(sbuffer* sbuf);
	void send_data(std::auto_ptr<vreflife> vbufife);

	// mp::wavy::handler interface
	void on_read(mp::wavy::event& e);

	void on_message(object msg, auto_zone z);

	void on_request(msgid_t msgid,
			object method, object params, auto_zone z)
	{
		throw msgpack::type_error();  // FIXME
	}

	void on_notify(
			object method, object params, auto_zone z)
	{
		throw msgpack::type_error();  // FIXME
	}

	void on_response(msgid_t msgid,
			object result, object error, auto_zone z)
	{
		throw msgpack::type_error();  // FIXME
	}
        void on_connection_closed_error() { } // do nothing
        void on_system_error(int system_errno ) { } // do nothing

protected:
	unpacker m_pac;
	loop m_loop;
};

template <typename MixIn>
inline stream_handler<MixIn>::stream_handler(int fd, loop lo) :
	mp::wavy::handler(fd),
	m_pac(MSGPACK_RPC_STREAM_BUFFER_SIZE),
	m_loop(lo) { }

template <typename MixIn>
inline stream_handler<MixIn>::~stream_handler() { }

template <typename MixIn>
inline void stream_handler<MixIn>::remove_handler()
{
	m_loop->remove_handler(fd());
}

template <typename MixIn>
inline void stream_handler<MixIn>::send_data(msgpack::sbuffer* sbuf)
{
	m_loop->write(fd(), sbuf->data(), sbuf->size(), &::free, sbuf->data());
	sbuf->release();
}

template <typename MixIn>
inline void stream_handler<MixIn>::send_data(std::auto_ptr<vreflife> vbuf)
{
	m_loop->writev(fd(), vbuf->vector(), vbuf->vector_size(), vbuf);
}

template <typename MixIn>
void stream_handler<MixIn>::on_message(object msg, auto_zone z)
{
	msg_rpc rpc;
	msg.convert(&rpc);

	switch(rpc.type) {
	case REQUEST: {
			msg_request<object, object> req;
			msg.convert(&req);
			static_cast<MixIn*>(this)->on_request(
					req.msgid, req.method, req.param, z);
		}
		break;

	case RESPONSE: {
			msg_response<object, object> res;
			msg.convert(&res);
			static_cast<MixIn*>(this)->on_response(
					res.msgid, res.result, res.error, z);
		}
		break;

	case NOTIFY: {
			msg_notify<object, object> req;
			msg.convert(&req);
			static_cast<MixIn*>(this)->on_notify(
					req.method, req.param, z);
		}
		break;

	default:
		throw msgpack::type_error();
	}
}

template <typename MixIn>
void stream_handler<MixIn>::on_read(mp::wavy::event& e)
try {
	while(true) {
		if(m_pac.execute()) {
			object msg = m_pac.data();
			auto_zone z( m_pac.release_zone() );
			m_pac.reset();

			//if(m_pac.nonparsed_size() > 0) {
			//	e.more();
			//} else {
			//	e.next();
			//}
			//stream_handler<MixIn>::on_message(msg, z);
			//return;

			// FIXME
			stream_handler<MixIn>::on_message(msg, z);
			if(m_pac.nonparsed_size() > 0) {
				continue;
			}
		}

		m_pac.reserve_buffer(MSGPACK_RPC_STREAM_RESERVE_SIZE);

		ssize_t rl = ::read(ident(), m_pac.buffer(), m_pac.buffer_capacity());
		if(rl <= 0) {
			if(rl == 0) { throw closed_exception(); }
			if(errno == EAGAIN || errno == EINTR) { return; }
			else { throw mp::system_error(errno, "read error"); }
		}

		m_pac.buffer_consumed(rl);
	}

} catch(msgpack::type_error& ex) {
	e.remove();
	return;
} catch(closed_exception& ex) {
  static_cast<MixIn*>(this)->on_connection_closed_error();
  e.remove();
  return;
} catch(mp::system_error &ex) {
  static_cast<MixIn*>(this)->on_system_error( ex.code );
  e.remove();
  return;
} catch(std::exception& ex) {
	e.remove();
	return;
} catch(...) {
	e.remove();
	return;
}

template <typename MixIn>
mp::shared_ptr<message_sendable> inline stream_handler<MixIn>::get_response_sender()
{
	return shared_self<stream_handler<MixIn> >();
}
// END copied code.

namespace socket {

class client_socket : public stream_handler<client_socket> {
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
            throw closed_exception();
        }
        s->on_response(msgid, result, error, z);
    }

private:
    weak_session session;

};

class client_transport: public rpc::client_transport {
    session_impl *session;
    mp::shared_ptr<client_socket> socket;
    GSocket *g_socket;
public:

    client_transport(session_impl* s, GSocket *socket) {
        int fd = g_socket_get_fd(socket);
        this->socket = s->get_loop_ref()->add_handler<client_socket>(fd, s);
        this->session = s;
        this->g_socket = socket;
        g_object_ref(socket);
    }

    ~client_transport() {
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

class server_socket : public stream_handler<server_socket> {
public:
	server_socket(int fd, shared_server svr);
	~server_socket();

	void on_request(
			msgid_t msgid,
			object method, object params, auto_zone z);

	void on_notify(
			object method, object params, auto_zone z);

private:
	weak_server m_svr;

private:
	server_socket();
	server_socket(const server_socket&);
};

server_socket::server_socket(int fd, shared_server svr) :
	stream_handler<server_socket>(fd, svr->get_loop_ref()),
	m_svr(svr) { }

server_socket::~server_socket() { }

void server_socket::on_request(
		msgid_t msgid,
		object method, object params, auto_zone z)
{
	shared_server svr = m_svr.lock();
	if(!svr) {
		throw closed_exception();
	}
	svr->on_request(get_response_sender(), msgid, method, params, z);
}

void server_socket::on_notify(
		object method, object params, auto_zone z)
{
	shared_server svr = m_svr.lock();
	if(!svr) {
		throw closed_exception();
	}
	svr->on_notify(method, params, z);
}

class server_transport : public rpc::server_transport {
public:
	server_transport(server_impl* svr, GSocket *socket);
	~server_transport();

  void close() { }
};

server_transport::server_transport(server_impl* svr, GSocket *socket)
{
    int fd = g_socket_get_fd(socket);
    weak_server wsvr = weak_server(mp::static_pointer_cast<server_impl>(
                svr->shared_from_this()));
    shared_server ssvr = wsvr.lock();
    ssvr->get_loop_ref()->add_handler<server_socket>(fd, ssvr);
    // TODO I think there is something missing here.
}

server_transport::~server_transport() { }

} // namespace socket
} // namespace transport

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
            new transport::socket::client_transport(s, this->socket));
}

socket_listener::socket_listener(GSocket *socket) {
    this->socket = socket;
    g_object_ref(this->socket);
}

socket_listener::~socket_listener() {
    g_object_unref(this->socket);
}

std::auto_ptr<server_transport>
socket_listener::listen(server_impl* s) const {
    return std::auto_ptr<server_transport>(
            new transport::socket::server_transport(s, this->socket));
}

} // namespace rpc
} // namespace msgpack
