
SELECT json_build_object(
               'name', user_groups0_name  ,'id', user_groups0_id   ,
               'users', (
                   SELECT json_agg(json_build_object(
                           'name', users_name1   ,
                           'tickets_aggregate', (
                               SELECT json_agg(json_build_object(
                                       'sum',
                                       json_build_object(  'number', users2_number_sum  )

                                   )) FROM (
                                               SELECT
                                                   sum(tickets2.number) as users2_number_sum
                                               from   tickets tickets2
                                               Where tickets2.creator_id =root.users1_id
                                           ) as root
                           )

                       )) FROM (
                                   SELECT
                                       users1.id as users1_id  ,users1.group_id as users1_group_id  ,users1.name as users_name1
                                   from   users users1
                                   Where users1.group_id =root.user_groups0_id  ORDER BY users1.created_at desc
                               ) as root
               )

           ) FROM (
                      SELECT
                          user_groups0.id as user_groups0_id  ,user_groups0.name as user_groups0_name
                      from   user_groups user_groups0
                      Where user_groups0.id = 'cfaa7e38-0a78-44e1-8e22-bb3a2733e579'  ORDER BY user_groups0.created_at desc
                  ) as root
