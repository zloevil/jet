create table test ( json String ) ENGINE = Test kafka_broker_list = '{{clickhouse_engines_kafka_resources_broker_list}}',
                          kafka_topic_list = '{{clickhouse_engines_kafka_resources_topic_list}}',
                          kafka_group_name = '{{clickhouse_engines_kafka_resources_group_name}}',
                          kafka_num_consumers = {{clickhouse_engines_kafka_resources_num_consumers}},
                          kafka_format = 'JSONAsString';