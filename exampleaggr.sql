SELECT json_build_object(
               'sum',
               json_build_object(  'number', sum(tickets0_number)  )
           ,'priority', tickets0_priority   ,
               'creator', (
                   SELECT json_build_object(
                                  'name', users_name1
                              ) FROM (
                                         SELECT
                                             users1.id as users1_id  ,users1.group_id as users1_group_id  ,users1.name as users_name1
                                         from   users users1
                                         Where users1.id =root.tickets0_creator_id
                                     ) as root
               )

           ) FROM (
                      SELECT
                          tickets0.id as tickets0_id  ,tickets0.creator_id as tickets0_creator_id  ,tickets0.tenant_id as tickets0_tenant_id  ,tickets0.number as tickets0_number  ,tickets0.priority as tickets0_priority
                      from   tickets tickets0  ORDER BY tickets0.created_at desc
                  ) as root
GROUP BY  tickets0_priority  ,tickets0_creator_id
