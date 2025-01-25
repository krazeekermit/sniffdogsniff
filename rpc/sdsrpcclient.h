#ifndef SDSRPCCLIENT_H
#define SDSRPCCLIENT_H

#include "sds_core/sds_config.h"
#include "sds_core/searchentriesdb.h"

#include <map>

class SdsRpcClient
{
public:
    SdsRpcClient(std::string nodeAddress_, SdsConfig cfg_ = {});

    int ping(const KadId &id, std::string address);
    int findNode(std::map<KadId, std::string> &nearest, const KadId &id);
    int storeResult(SearchEntry se);
    int findResults(std::vector<SearchEntry> &results, const char *query);

private:
    int newConnection();
    int sendRpcRequest(uint8_t fun, SdsBytesBuf &args, SdsBytesBuf &reply);

    std::string nodeAddress;
    SdsConfig config;
};

#endif // SDSRPCCLIENT_H
