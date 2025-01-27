#include "sdsrpcclient.h"

#include "rpc_common.h"

#include "net/socks5.h"
#include "net/libsam3.h"
#include "net/netutil.h"
#include "common/logging.h"

#include <arpa/inet.h>
#include <sys/socket.h>
#include <unistd.h>

SdsRpcClient::SdsRpcClient(std::string nodeAddress_, SdsConfig cfg_)
    : nodeAddress(nodeAddress_), config(cfg_)
{

}

int SdsRpcClient::ping(const KadId &id, std::string address)
{
    SdsBytesBuf a, r;
    PingArgs args(id, address);
    args.write(a);

    return sendRpcRequest(FUNC_PING, a, r);
}

int SdsRpcClient::findNode(std::map<KadId, std::string> &nearest, const KadId &callerId, std::string callerAddress, const KadId &id)
{
    SdsBytesBuf a, r;

    FindNodeArgs args(callerId, callerAddress, id);
    args.write(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != 0)
        return ret;

    FindNodeReply reply;
    reply.read(r);

    nearest = reply.nearest;
    return ERR_NULL;
}

int SdsRpcClient::storeResult(const KadId &callerId, std::string callerAddress, SearchEntry se)
{
    SdsBytesBuf a, r;

    StoreResultArgs args(callerId, callerAddress, se);
    args.write(a);

    return sendRpcRequest(FUNC_STORE_RESULT, a, r);
}

int SdsRpcClient::findResults(std::vector<SearchEntry> &results, const KadId &callerId, std::string callerAddress, const char *query)
{
    SdsBytesBuf a, r;

    FindResultsArgs args(callerId, callerAddress, query);
    args.write(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != 0)
        return ret;

    FindResultsReply reply;
    reply.read(r);

    results = reply.results;
    return ERR_NULL;
}

int SdsRpcClient::newConnection()
{
    int port = 0;
    int fd = -1;

    char suffix[64];
    char addr[1024];
    if (net_urlparse(addr, suffix, &port, this->nodeAddress.c_str())) {
        return -1;
    }
    if (strcmp(suffix, ".onion") == 0 || this->config.force_tor_proxy) {
        fd = socks5_connect(this->config.tor_socks5_addr, this->config.tor_socks5_port, addr, port);
        if (fd < 1) {
            logdebug << "error connecting to socks5 socket: " << socks5_strerror(fd);
            return -1;
        }
    } else if (strcmp(suffix, ".i2p") == 0) {
        Sam3Session ses;
        Sam3Connection *samConn = nullptr;
        if (sam3CreateSession(&ses, this->config.i2p_sam_addr, this->config.i2p_sam_port, nullptr, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
            return -2;
        }
        if (strstr(addr, ".b32.i2p")) {
            if (sam3NameLookup(&ses, this->config.i2p_sam_addr, this->config.i2p_sam_port, addr)) {
                logdebug << "i2p naming lookup fail:" << ses.error;
                sam3CloseSession(&ses);
                return -2;
            }
            samConn = sam3StreamConnect(&ses, ses.destkey);
        } else {
            samConn = sam3StreamConnect(&ses, addr);
        }
        if (!samConn) {
            sam3CloseSession(&ses);
            return -2;
        }

        fd = samConn->fd;
    } else {
        fd = net_socket_connect(addr, port, 5);
        if (fd <= 0) {
            logerr << "error connecting to " << this->nodeAddress << ": " << strerror(errno);
        }
    }

    return fd;
}

static void fillRandIdVec(uint8_t *vec, size_t sz)
{
    FILE *rfp = fopen("/dev/urandom", "rb");
    if (!rfp)
        return;

    fread(vec, sizeof(uint8_t), sz, rfp);
    fclose(rfp);
}

int SdsRpcClient::sendRpcRequest(uint8_t funcode, SdsBytesBuf &args, SdsBytesBuf &reply)
{
    int fd = this->newConnection();
    if (fd < 0)
        return -1;

    /* Send Request */
    RpcRequestHeader req = {
        .funcode = funcode,
        .datasize = args.size()
    };
    fillRandIdVec(req.id, ID_SIZE);

    int errcode = ERR_NULL;

    RpcResponseHeader resp;
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

    if (memcmp(resp.id, req.id, ID_SIZE)) {
        errcode = ERR_REQ_ID;
        goto rpc_fail;
    }

    if (resp.datasize > 0) {
        reply.allocate(resp.datasize);
        if (recv(fd, reply.bufPtr(), resp.datasize, 0) != resp.datasize) {
            errcode = ERR_RECV_REQUEST;
            goto rpc_fail;
        }
    }

rpc_fail:

    close(fd);
    return errcode;
}
