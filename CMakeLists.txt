cmake_minimum_required(VERSION 3.5)

project(sniffdogsniffd LANGUAGES C CXX)

link_libraries(pthread crypto db-5.3 gumbo curl)

set(CMAKE_CXX_STANDARD 14)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

if (SDS_TESTING)
    enable_testing()

    add_executable(sniffdogsniffd test.cpp)
    target_link_libraries(sniffdogsniffd gtest)

    add_test(sniffdogsniffd sniffdogsniffd)
else()
    add_executable(sniffdogsniffd main.cpp)
endif()

# ArchLinux and derivates are putting the BerkleyDB include file in a different directory
target_include_directories(sniffdogsniffd PRIVATE "/usr/include/db5.3")

target_include_directories(sniffdogsniffd PRIVATE "${PROJECT_SOURCE_DIR}")
add_subdirectory(common)
add_subdirectory(sds_core)
add_subdirectory(p2p)
add_subdirectory(kademlia)
add_subdirectory(crawler)
add_subdirectory(net)
add_subdirectory(webserver)

include(GNUInstallDirs)
install(TARGETS sniffdogsniffd
    LIBRARY DESTINATION ${CMAKE_INSTALL_LIBDIR}
    RUNTIME DESTINATION ${CMAKE_INSTALL_BINDIR}
)
