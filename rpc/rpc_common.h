#ifndef RPC_COMMON_H
#define RPC_COMMON_H

#include "macros.h"
#include "kademlia/kadbucket.h"
#include "searchentriesdb.h"
#include "sdsbytesbuf.h"

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
    Ping
*/
struct PingArgs {
    KadId id;
    std::string address;

    PingArgs();
    PingArgs(const KadId &id_, std::string address_);

    int read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct PingReply {
};

/*
    FindNode
*/
struct FindNodeArgs {
    KadId id;

    FindNodeArgs();
    FindNodeArgs(const KadId &id_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct FindNodeReply {
    std::map<KadId, std::string> nearest;

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

/*
    StoreResult
*/
struct StoreResultArgs {
    SearchEntry se;

    StoreResultArgs() = default;
    StoreResultArgs(SearchEntry se_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct StoreResultReply {
};

/*
    FindResults
*/
struct FindResultsArgs {
    std::string query;

    FindResultsArgs() = default;
    FindResultsArgs(std::string query_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct FindResultsReply {
    std::vector<SearchEntry> results;

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

#endif // RPC_COMMON_H
