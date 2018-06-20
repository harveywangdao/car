#!/usr/bin/python3

import pymysql
import time

dbip = "localhost"
user = "root"
password = "180498"

dbname = "iotdb"
tablename = "thingbaseinfodata_tbl"

# dbname = "thingsdb"
# tablename = "thingbaseinfodata_tbl"

db = pymysql.connect(dbip,user,password,dbname)

#truncate table
cursor = db.cursor()
cursor.execute("truncate table %s" % tablename)
cursor.close()

vin=1000000

begin_time = time.time()

def generate_data(num):
    return (
        '12345678901234567890123456789',
        '1234567890123456',
        'WDDUX5268F'+str(num),
        '12345678901234567890',
        'HGTSN12584UHGTE',
        0,
        0,
        '1234567890123456',
        0
    )

step = 1000
times = 100

try:
    print("Start insert data...")
    column_count = 9
    insert_sql = "INSERT INTO "+ tablename +" (thingserialno,prethingaes128key,thingid,iccid,imsi,status,bid,thingaes128key,eventcreationtime) \
    VALUES (" + ",".join([ "%s" for x in range(column_count)]) + ")"
    for x in range(times):
        print("vin = %d." % vin)

        batch_data = [generate_data(vin+x) for x in range(step)]
        vin+=step

        cursor = db.cursor()
        try:
            cursor.executemany(insert_sql, batch_data)
        except Exception as e:
            db.rollback()
            print("rollback db"+str(e))
            raise
        else:
            db.commit()
            print('commit data.')
        finally:
            cursor.close()

except Exception as e:
    print(str(e))
    raise
else:
    pass
finally:
    pass

end_time = time.time()
total_sec = end_time - begin_time
print ("总共耗时(s): " + str(total_sec))

db.close()