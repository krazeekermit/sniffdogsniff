* Unify code for creating socket in net/netutil.c with the following:
    - Add timeout (select)
    - Replace socket creation with unified code everywere
    - better error handling

* Add crawler
* Add SIGINT handling
* Add self node id creation based on cfg
* Add manpages for cli args and for config file

* Investigate replace Cmake with plain makefiles
