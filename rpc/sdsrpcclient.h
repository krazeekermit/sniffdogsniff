#ifndef SDSRPCCLIENT_H
#define SDSRPCCLIENT_H

#include "rpc_common.h"
#include "sds_core/sds_config.h"
#include "sds_core/searchentriesdb.h"

#include <map>

class SdsRpcClient
{
public:
    SdsRpcClient(SdsConfig &cfg_, std::string nodeAddress_);

    int ping(const KadId &id, std::string address);
    int findNode(FindNodeReply &reply, const KadId &callerId, std::string callerAddress, const KadId &id);
    int storeResult(const KadId &callerId, std::string callerAddress, SearchEntry se);
    int findResults(FindResultsReply &reply, const KadId &callerId, std::string callerAddress, const char *query);

private:
    int newConnection();
    int sendRpcRequest(uint8_t fun, SdsBytesBuf &args, SdsBytesBuf &reply);

    SdsConfig config;
    std::string nodeAddress;
};

#endif // SDSRPCCLIENT_H
