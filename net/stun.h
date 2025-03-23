#ifndef STUN_H
#define STUN_H

// Incomplete: neded for Hole-Punching for NAT traversal
int stun_bind_request(const char *stun_addr, int stun_port, struct sockaddr_in *reflexive_addr);

#endif // STUN_H
