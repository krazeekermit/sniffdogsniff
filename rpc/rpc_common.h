#ifndef RPC_COMMON_H
#define RPC_COMMON_H

#include "common/sdsbytesbuf.h"
#include "sds_core/searchentriesdb.h"

#include <cstdint>
#include <cstring>
#include <map>

#define ID_SIZE           24

/* Errors Codes */
#define ERR_NULL          0
#define ERR_RECV_REQUEST  1
#define ERR_REQ_ID        2
#define ERR_SERIALIZE     3
#define ERR_NOFUNCT       4
#define ERR_TYPE_ARGUMENT 5
#define ERR_CONNECTION    6

/* Function Codes */
#define FUNC_PING         100
#define FUNC_FIND_NODE    101
#define FUNC_STORE_RESULT 102
#define FUNC_FIND_RESULTS 103

static const char *rpc_strfunction(int fun)
{
    switch (fun) {
    case FUNC_PING:
        return "FUNC_PING";
    case FUNC_FIND_NODE:
        return "FUNC_FIND_NODE";
    case FUNC_STORE_RESULT:
        return "FUNC_STORE_RESULT";
    case FUNC_FIND_RESULTS:
        return "FUNC_FIND_RESULTS";
    }
    return "";
}

static const char *rpc_strerror(int err)
{
    switch (err) {
    case ERR_RECV_REQUEST:
        return "corrupted or bad request";
    case ERR_NOFUNCT:
        return "function does not exists";
    case ERR_SERIALIZE:
        return "error packing/unpacking function arguments";
    case ERR_TYPE_ARGUMENT:
        return "wrong arguments for function";
    case ERR_CONNECTION:
        return "unable to connect";
    }
    return "";
}

packed_struct RpcRequestHeader {
    uint8_t funcode;
    uint8_t id[ID_SIZE];
    uint64_t datasize;
};

packed_struct RpcResponseHeader {
    uint8_t funcode;
    uint8_t errcode;
    uint8_t id[ID_SIZE];
    uint64_t datasize;
};

/*
    RpcReplyException
*/

class SdsRpcException : public std::runtime_error {
public:
    SdsRpcException(int errcode_)
        : std::runtime_error("p2p rpc exception: " + std::string(rpc_strerror(errcode_)))
    {}
};

/*
    Rpc Args
*/
struct ArgsBase
{
    KadId callerId;
    std::string callerAddress;

    ArgsBase() = default;
    ArgsBase(const KadId &id_, std::string address_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct ReplyBase
{
};

/*
    Ping
*/
struct PingArgs : public ArgsBase
{

    PingArgs() = default;
    PingArgs(const KadId &callerId_, std::string callerAddress_);
};

struct PingReply : public ReplyBase
{
};

/*
    FindNode
*/
struct FindNodeArgs : public ArgsBase
{
    KadId targetId;

    FindNodeArgs() = default;
    FindNodeArgs(const KadId &callerId_, std::string callerAddress_, const KadId &targetId_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct FindNodeReply : public ReplyBase
{
    std::map<KadId, std::string> nearest;

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

/*
    StoreResult
*/
struct StoreResultArgs : public ArgsBase
{
    SearchEntry se;

    StoreResultArgs() = default;
    StoreResultArgs(const KadId &callerId_, std::string callerAddress_, SearchEntry se_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct StoreResultReply : public ReplyBase
{
};

/*
    FindResults
*/
struct FindResultsArgs : public ArgsBase
{
    std::string query;

    FindResultsArgs() = default;
    FindResultsArgs(const KadId &callerId_, std::string callerAddress_, std::string query_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct FindResultsReply : public FindNodeReply
{
    std::vector<SearchEntry> results;

    bool hasResults();

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

#endif // RPC_COMMON_H
