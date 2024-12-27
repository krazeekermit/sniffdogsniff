#ifndef RPC_COMMON_H
#define RPC_COMMON_H

#include "macros.h"
#include "kademlia/kadbucket.h"
#include "searchentriesdb.h"
#include "msgpack11/msgpack11.h"

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

    PingArgs()
        : address("")
    {
        memset(this->id.id, 0, KAD_ID_SZ);
    }

    PingArgs(const KadId &id_, std::string address_)
        : address(address_), id(id_)
    {}

    bool unpack(msgpack11::MsgPack &obj)
    {
        if (!obj.is_array())
            return false;

        if (!obj[0].is_string())
            return false;

        memcpy(this->id.id, obj[0].string_value().c_str(), KAD_ID_SZ);

        if (!obj[1].is_string())
            return false;

        address = obj[1].string_value();

        return true;
    }

    void pack(msgpack11::MsgPack &obj)
    {
        obj = msgpack11::MsgPack::array {
            std::string(id.id, id.id + KAD_ID_SZ), address
        };
    }
};

struct PingReply {

};

/*
    FindNode
*/
struct FindNodeArgs {
    KadId id;

    FindNodeArgs()
    {
        memset(this->id.id, 0, KAD_ID_SZ);
    }

    FindNodeArgs(const KadId &id_)
        : id(id_)
    {}

    ~FindNodeArgs()
    {}

    bool unpack(msgpack11::MsgPack &obj)
    {
        if (!obj.is_array())
            return false;

        if (!obj[0].is_string())
            return false;

        memcpy(this->id.id, obj[0].string_value().c_str(), KAD_ID_SZ);

        return true;
    }

    void pack(msgpack11::MsgPack &obj)
    {
        obj = msgpack11::MsgPack::array {
            std::string(this->id.id, this->id.id + KAD_ID_SZ)
        };
    }
};

struct FindNodeReply {
    std::map<KadId, std::string> nearest;

    FindNodeReply()
    {}

    ~FindNodeReply()
    {}

    bool unpack(msgpack11::MsgPack &obj)
    {
        if (!obj.is_object())
            return false;

        nearest.clear();
        for (auto it = obj.object_items().begin(); it != obj.object_items().end(); it++) {
            KadId id;
            memcpy(id.id, it->first.string_value().c_str(), KAD_ID_SZ);
            nearest[id] = it->second.string_value();
        }

        return true;
    }

    void pack(msgpack11::MsgPack &obj)
    {
        msgpack11::MsgPack::object om = {};
        for (auto it = nearest.begin(); it != nearest.end(); it++) {
            std::string idstr(it->first.id, it->first.id + KAD_ID_SZ);
            om[idstr] = it->second;
        }
        obj = om;
    }
};

/*
    StoreResult
*/
struct StoreResultArgs {
    SearchEntry se;

    StoreResultArgs(SearchEntry se_)
        : se(se_)
    {}

    StoreResultArgs()
    {}

    bool unpack(msgpack11::MsgPack &obj)
    {
        return se.unpack(obj);
    }

    void pack(msgpack11::MsgPack &obj)
    {
        se.pack(obj);
    }
};

struct StoreResultReply {

};

/*
    FindResults
*/
struct FindResultsArgs {
    std::string query;

    FindResultsArgs(std::string query_)
        : query(query_)
    {}

    FindResultsArgs()
    {}

    bool unpack(msgpack11::MsgPack &obj)
    {
        if (!obj.is_string())
            return false;

        query = obj.string_value();
        return true;
    }

    void pack(msgpack11::MsgPack &obj)
    {
        obj = query;
    }
};

struct FindResultsReply {
    std::vector<SearchEntry> results;

    bool unpack(msgpack11::MsgPack &obj)
    {
        results.clear();
        if (!obj.is_array())
            return false;

        msgpack11::MsgPack::array a = obj.array_items();
        for (auto it = a.begin(); it < a.end(); it++) {
            SearchEntry se;
            if (!se.unpack(*it))
                return false;

            results.push_back(se);
        }
        return true;
    }

    void pack(msgpack11::MsgPack &obj)
    {
        msgpack11::MsgPack::array a;
        for (auto it = results.begin(); it < results.end(); it++) {
            msgpack11::MsgPack o;
            it->pack(o);
            a.push_back(o);
        }
        obj = a;
    }
};

#endif // RPC_COMMON_H
