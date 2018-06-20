#!/usr/bin/python
# -*- coding: UTF-8 -*-

import MySQLdb
import MySQLdb.cursors
import random
import string
import time

#批量插的次数
loop_count = 1000000

#每次批量查的数据量
batch_size = 100
success_count = 0
fails_count = 0

conn = MySQLdb.connect(host="127.0.0.1", user="root", passwd="123456", db="example", port=3306, cursorclass = MySQLdb.cursors.SSCursor)
chars = 'AaBbCcDdEeFfGgHhIiJjKkLlMmNnOoPpQqRrSsTtUuVvWwXxYyZz'
digits = '0123456789'

def random_generate_string(length):
    return string.join(random.sample(chars, length), '')

def random_generate_number(length):
    if length > len(digits):
        digit_list = random.sample(digits, len(digits))
        digit_list.append(random.choice(digits))
        return string.join(digit_list, '')
    return string.join(random.sample(digits, length), '')

def random_generate_data(num):
    c = [num]
    phone_num_seed = 13100000000
    def _random_generate_data():
        c[0] += 1
        return (
            c[0],
            "last_name_" + str(random.randrange(100000)),
            "first_name_" + str(random.randrange(100000)),
            random.choice('MF'),
            random.randint(1, 120),
            phone_num_seed + c[0],
            random_generate_string(20),
            random_generate_string(10),
            time.strftime("%Y-%m-%d %H:%M:%S")
        )
    return _random_generate_data

def execute_many(insert_sql, batch_data):
    global success_count, fails_count
    cursor = conn.cursor()
    try:
        cursor.executemany(insert_sql, batch_data)
    except Exception, e:
        conn.rollback()
        fails_count = fails_count + len(batch_data)
        print e
        raise
    else:
        conn.commit()
        success_count = success_count + len(batch_data)
        print str(success_count) + " commit"
    finally:
        cursor.close()
try:
    #user表列的数量
    column_count = 9

    insert_sql = "replace into user(id, last_name, first_name, sex, age, phone, address, password, create_time) values (" + ",".join([ "%s" for x in xrange(column_count)]) + ")"
    batch_count = 0
    begin_time = time.time()
    for x in xrange(loop_count):
        batch_count =  x * batch_size
        gen_fun = random_generate_data(batch_count)
        batch_data = [gen_fun() for x in xrange(batch_size)]
        execute_many(insert_sql, batch_data)
    end_time = time.time()
    total_sec = end_time - begin_time
    qps = success_count / total_sec
    print "总共生成数据： " + str(success_count)
    print "总共耗时(s): " + str(total_sec)
    print "QPS: " + str(qps)
except Exception, e:
    print e
    raise
else:
    pass
finally:
    pass