#!/usr/bin/sh

sudo rm -Rf my_pgdata
cp db/01_create_database.sql.example db/01_create_database.sql
cp db/03_init_data.sql.example db/03_init_data.sql
