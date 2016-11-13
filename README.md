# mysql-query-logger

Ever want to log all the SQL statements being executed against a MySQL
database, but don't want to crush your DB server like an empty beer can
under the I/O load of all that logging? mysql-query-logger is a program
which logs SQL queries on a MySQL server by monitoring the local network
and picking out the packets that look like queries.

## Usage

```
mysql-query-logger [-h host] [-p port] [-i interface] [logfile]
```

_host_ and _port_ are the host and port that your MySQL server is accepting
connections on. _interface_ is the local network interface. If no _logfile_
is supplied, this program logs to standard output.

For obvious reasons, this program needs to be run on a machine in the same
subnet as the MySQL server; if it's a switched network, you'll have to
configure your switch to tee the MySQL server's traffic to the server
running mysql-query-logger.

## License

Distributed under the MIT License. See the LICENSE file included in this
repository for details.
