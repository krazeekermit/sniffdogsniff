* Unify code for creating socket in net/netutil.c with the following:
    - Add timeout (select)
    - Replace socket creation with unified code everywere
    - better error handling
