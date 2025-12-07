#ifndef SDSP2PCLIENT_H
#define SDSP2PCLIENT_H

#include "p2p_common.h"
#include "sds_core/sdsconfigfile.h"
#include "sds_core/searchentriesdb.h"

#include <map>

class SdsP2PClient
{
public:
    SdsP2PClient(SdsConfigFile *configFile, std::string nodeAddress_);

    int ping(const KadId &id, std::string address);
    int findNode(FindNodeReply &reply, const KadId &callerId, std::string callerAddress, const KadId &id);
    int storeResult(const KadId &callerId, std::string callerAddress, SearchEntry se);
    int findResults(FindResultsReply &reply, const KadId &callerId, std::string callerAddress, const char *query);

private:
    int newConnection();
    int sendRpcRequest(uint8_t fun, SdsBytesBuf &args, SdsBytesBuf &reply);

    /* Network configurations */
    // TOR
    bool forceTorProxy;
    std::string torSocks5Addr;
    int torSocks5Port;

    // I2P
    std::string i2pSamAddr;
    int i2pSamPort;

    std::string nodeAddress;
};

#endif // SDSP2PCLIENT_H
