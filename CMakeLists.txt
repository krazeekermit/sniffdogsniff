cmake_minimum_required(VERSION 3.5)

project(sniffdogsniffd LANGUAGES C CXX)

link_libraries(pthread crypto db gumbo curl)

set(CMAKE_CXX_STANDARD 11)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_executable(sniffdogsniffd
    main.cpp
    utils.h macros.h
    logging.h logging.cpp
    simhash.h simhash.cpp
    sds_config.h sds_config.cpp
    sdstask.h sdstask.cpp
    localnode.h localnode.cpp
    searchentriesdb.h searchentriesdb.cpp
    searchengine.h searchengine.cpp
    )

target_include_directories(sniffdogsniffd PRIVATE "${PROJECT_SOURCE_DIR}")
add_subdirectory(msgpack11)
add_subdirectory(rpc)
add_subdirectory(kademlia)
add_subdirectory(crawler)
add_subdirectory(net)
add_subdirectory(webserver)

include(GNUInstallDirs)
install(TARGETS sniffdogsniffd
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
