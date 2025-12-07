#include "sdsp2pclient.h"

#include "p2p_common.h"

#include "net/socks5.h"
#include "net/libsam3.h"
#include "net/netutil.h"

#include <arpa/inet.h>
#include <sys/socket.h>
#include <unistd.h>
#include <climits>

SdsP2PClient::SdsP2PClient(SdsConfigFile *configFile, std::string nodeAddress_)
    :   forceTorProxy(configFile->getDefaultSection()->lookupBool("force_tor_proxy")),
        torSocks5Addr(configFile->getDefaultSection()->lookupString("tor_socks5_addr")),
        torSocks5Port(configFile->getDefaultSection()->lookupInt("tor_socks5_port")),
        i2pSamAddr(configFile->getDefaultSection()->lookupString("i2p_sam_addr")),
        i2pSamPort(configFile->getDefaultSection()->lookupInt("i2p_sam_port")),
        nodeAddress(nodeAddress_)
{}

int SdsP2PClient::ping(const KadId &id, std::string address)
{
    SdsBytesBuf a, r;
    PingArgs args(id, address);
    args.write(a);

    return sendRpcRequest(FUNC_PING, a, r);
}

int SdsP2PClient::findNode(FindNodeReply &reply, const KadId &callerId, std::string callerAddress, const KadId &id)
{
    SdsBytesBuf a, r;

    FindNodeArgs args(callerId, callerAddress, id);
    args.write(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != ERR_NULL)
        throw SdsP2PException(ret);

    reply.read(r);
    return ERR_NULL;
}

int SdsP2PClient::storeResult(const KadId &callerId, std::string callerAddress, SearchEntry se)
{
    SdsBytesBuf a, r;

    StoreResultArgs args(callerId, callerAddress, se);
    args.write(a);

    return sendRpcRequest(FUNC_STORE_RESULT, a, r);
}

int SdsP2PClient::findResults(FindResultsReply &reply, const KadId &callerId, std::string callerAddress, const char *query)
{
    SdsBytesBuf a, r;

    FindResultsArgs args(callerId, callerAddress, query);
    args.write(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != 0) {
        throw SdsP2PException(ret);
    }

    reply.read(r);
    return ERR_NULL;
}

int SdsP2PClient::newConnection()
{
    int port = 0;
    int fd = -1;

    char suffix[64];
    char addr[1024];
    memset(suffix, 0, sizeof(suffix));
    memset(addr, 0, sizeof(addr));
    if (net_urlparse(addr, suffix, &port, this->nodeAddress.c_str())) {
        return -1;
    }
    if (strcmp(suffix, ".i2p") == 0) {
        Sam3Session ses;
        Sam3Connection *samConn = nullptr;
        if (sam3CreateSession(&ses, this->i2pSamAddr.c_str(), this->i2pSamPort, nullptr, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
            throw std::runtime_error("i2p sam session creation error: " + std::string(ses.error));
            return -2;
        }
        if (strstr(addr, ".b32.i2p")) {
            if (sam3NameLookup(&ses, this->i2pSamAddr.c_str(), this->i2pSamPort, addr)) {
                sam3CloseSession(&ses);
                throw std::runtime_error("i2p sam naming lookup error: " + std::string(ses.error));
                return -2;
            }
            samConn = sam3StreamConnect(&ses, ses.destkey);
        } else {
            samConn = sam3StreamConnect(&ses, addr);
        }
        if (!samConn) {
            sam3CloseSession(&ses);
            throw std::runtime_error("i2p sam session creation error: " + std::string(ses.error));
            return -2;
        }

        fd = samConn->fd;
    } else if (strcmp(suffix, ".onion") == 0 || this->forceTorProxy) {
        fd = socks5_connect(this->torSocks5Addr.c_str(), this->torSocks5Port, addr, port);
        if (fd < 1) {
            throw std::runtime_error("error connecting to socks5 socket: " + std::string(socks5_strerror(fd)));
            return -1;
        }
    } else {
        fd = net_socket_connect(addr, port, 3000);
        if (fd <= 0) {
            throw std::runtime_error("error connecting to " + this->nodeAddress + ": " + strerror(errno));
        }
    }

    return fd;
}

int SdsP2PClient::sendRpcRequest(uint8_t funcode, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    int fd = this->newConnection();

    /* Send Request */
    MessageRequestHeader req = {
        .funcode = funcode,
        .id = rand() % ULONG_MAX,
        .datasize = htole64(args.size())
    };

    int errcode = ERR_NULL;

    uint64_t resp_sz = 0;
    MessageResponseHeader resp;
    memset(&resp, 0, sizeof(resp));

    if (send(fd, &req, sizeof(req), 0) != sizeof(req)) {
        errcode = -2;
        goto rpc_fail;
    }
    if (send(fd, args.bufPtr(), args.size(), 0) != args.size()) {
        errcode = -2;
        goto rpc_fail;
    }

    /* Receive Reply */
    if (recv(fd, &resp, sizeof(resp), 0) != sizeof(resp)) {
        errcode = ERR_RECV_REQUEST;
        goto rpc_fail;
    }

    if (resp.errcode != ERR_NULL) {
        errcode = resp.errcode;
        goto rpc_fail;
    }

    if (resp.id != req.id) {
        errcode = ERR_REQ_ID;
        goto rpc_fail;
    }

    resp_sz = le64toh(resp.datasize);
    if (resp_sz > 0) {
        reply.allocate(resp_sz);
        if (recv(fd, reply.bufPtr(), resp_sz, 0) != resp_sz) {
            errcode = ERR_RECV_REQUEST;
            goto rpc_fail;
        }
    }

rpc_fail:

    close(fd);
    return errcode;
}
