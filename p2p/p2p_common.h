#ifndef P2P_COMMON_H
#define P2P_COMMON_H

#include "common/sdsbytesbuf.h"
#include "sds_core/searchentriesdb.h"

#include <cstdint>
#include <cstring>
#include <map>

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

static const char *p2p_strfunction(int fun)
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

static const char *p2p_strerror(int err)
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

packed_struct MessageRequestHeader {
    uint8_t funcode;
    uint64_t id;
    uint64_t datasize;
};

packed_struct MessageResponseHeader {
    uint8_t funcode;
    uint8_t errcode;
    uint64_t id;
    uint64_t datasize;
};

/*
    RpcReplyException
*/

class SdsP2PException : public std::runtime_error {
public:
    SdsP2PException(int errcode_)
        : std::runtime_error("p2p exception: " + std::string(p2p_strerror(errcode_)))
    {}
};

/*
    Ping
*/
struct PingArgs
{
    uint64_t nonce;
    KadId id;
    std::string address;

    PingArgs();
    PingArgs(const KadId &id_, std::string address_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct PingReply
{
    uint64_t nonce;

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

/*
    FindNode
*/
struct FindNodeArgs
{
    KadId targetId;

    FindNodeArgs() = default;
    FindNodeArgs(const KadId &targetId_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct FindNodeReply
{
    std::map<KadId, std::string> nearest;

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

/*
    StoreResult
*/
struct StoreResultArgs
{
    SearchEntry se;

    StoreResultArgs() = default;
    StoreResultArgs(SearchEntry se_);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);
};

struct StoreResultReply
{};

/*
    FindResults
*/
struct FindResultsArgs
{
    std::string query;

    FindResultsArgs() = default;
    FindResultsArgs(std::string query_);

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

#endif // P2P_COMMON_H
