create table test ( json String ) ENGINE = Test kafka_broker_list = 'broker',
                          kafka_topic_list = 'topic.1.1.2',
                          kafka_group_name = 'test.group',
                          kafka_num_consumers = 1,
                          kafka_format = 'JSONAsString';