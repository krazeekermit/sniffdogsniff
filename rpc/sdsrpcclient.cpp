#include "sdsrpcclient.h"

#include "rpc_common.h"

#include "net/socks5.h"
#include "net/libsam3.h"
#include "logging.hpp"

#include <arpa/inet.h>
#include <sys/socket.h>
#include <unistd.h>

SdsRpcClient::SdsRpcClient(std::string nodeAddress_, SdsConfig cfg_)
    : nodeAddress(nodeAddress_), config(cfg_)
{

}

int SdsRpcClient::ping(const KadId &id, std::string address)
{
    msgpack11::MsgPack a, r;
    PingArgs args(id, address);
    args.pack(a);

    return sendRpcRequest(FUNC_PING, a, r);
}

int SdsRpcClient::findNode(std::map<KadId, std::string> &nearest, const KadId &id)
{
    msgpack11::MsgPack a, r;

    FindNodeArgs args(id);
    args.pack(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != 0)
        return ret;

    FindNodeReply reply;
    if (!reply.unpack(r))
        return ERR_SERIALIZE;

    nearest = reply.nearest;
    return ERR_NULL;
}

int SdsRpcClient::storeResult(SearchEntry se)
{
    msgpack11::MsgPack a, r;

    StoreResultArgs args(se);
    args.pack(a);

    return sendRpcRequest(FUNC_STORE_RESULT, a, r);
}

int SdsRpcClient::findResults(std::vector<SearchEntry> &results, const char *query)
{
    msgpack11::MsgPack a, r;

    FindResultsArgs args(query);
    args.pack(a);

    int ret = sendRpcRequest(FUNC_FIND_NODE, a, r);
    if (ret != 0)
        return ret;

    FindResultsReply reply;
    if (!reply.unpack(r))
        return ERR_SERIALIZE;

    results = reply.results;
    return ERR_NULL;
}

int SdsRpcClient::newConnection()
{
    int port = 0;
    int fd = -1;

    char *sxp = nullptr;
    char naBuf[512];
    strncpy(naBuf, this->nodeAddress.c_str(), this->nodeAddress.length());
    char *addr = strtok_r(naBuf, ":", &sxp);

    if (sxp)
        port = atoi(sxp);

    char *suffix = strrchr(addr, '.');
    if (suffix && (strcmp(suffix, ".onion") == 0 || this->config.force_tor_proxy)) {
        fd = socks5_connect(this->config.tor_socks5_addr, this->config.tor_socks5_port, addr, port);
        if (fd < 1) {
            logdebug(<< "error connecting to socks5 socket: " << socks5_strerror(fd));
            return -1;
        }
    } else if (suffix && strcmp(suffix, ".i2p") == 0) {
        Sam3Session ses;
        if (sam3NameLookup(&ses, this->config.i2p_sam_addr, this->config.i2p_sam_port, addr)) {
            logdebug(<< "i2p naming lookup fail:" << ses.error);
            return -2;
        }
        if (sam3CreateSession(&ses, this->config.i2p_sam_addr, this->config.i2p_sam_port, nullptr, Sam3SessionType::SAM3_SESSION_STREAM, Sam3SigType::EdDSA_SHA512_Ed25519, nullptr)) {
            return -2;
        }
        Sam3Connection *samConn = sam3StreamConnect(&ses, ses.destkey);
        if (!samConn) {
            return -2;
        }

        fd = samConn->fd;
    } else {
        int i;
        ssize_t valread;
        struct sockaddr_in address;
        int opt = 1;
        socklen_t addrlen = sizeof(address);

        fd = socket(AF_INET, SOCK_STREAM, 0);
        if (fd < 0)
            return -1;

        if (inet_pton(AF_INET, addr, &address.sin_addr) <= 0) {
            return -1;
        }

        address.sin_family = AF_INET;
        address.sin_port = htons(port);

        if (connect(fd, (struct sockaddr*)&address, sizeof(address)) < 0) {
            return -2;
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

int SdsRpcClient::sendRpcRequest(uint8_t funcode, msgpack11::MsgPack &args, msgpack11::MsgPack &reply)
{
    int fd = this->newConnection();
    if (fd < 0)
        return -1;

    /* Send Request */
    std::string argsDump = args.dump();
    RpcRequestHeader req = {
        .funcode = funcode,
        .datasize = argsDump.length()
    };
    fillRandIdVec(req.id, ID_SIZE);

    int errcode = ERR_NULL;
    char *rcvbuf = nullptr;
    std::string unpackerrstr = "";

    RpcResponseHeader resp;
    memset(&resp, 0, sizeof(resp));

    if (send(fd, &req, sizeof(req), 0) != sizeof(req)) {
        errcode = -2;
        goto rpc_fail;
    }
    if (send(fd, argsDump.c_str(), argsDump.length(), 0) != argsDump.length()) {
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

    rcvbuf = new char[resp.datasize];
    if (recv(fd, rcvbuf, resp.datasize, 0) != resp.datasize) {
        errcode = ERR_RECV_REQUEST;
        goto rpc_fail;
    }

    reply = msgpack11::MsgPack::parse(rcvbuf, resp.datasize, unpackerrstr);

rpc_fail:

    close(fd);
    delete[] rcvbuf;
    return errcode;
}
