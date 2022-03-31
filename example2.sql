SELECT json_build_object(
    'name', tickets0_name  ,'id', tickets0_id  ,'creator_id', tickets0_creator_id   ,
   'categories', (
    SELECT json_agg(json_build_object(
    'name', categories_name1  ,'id', categories_id1   ,
   'creator', (
   SELECT json_build_object(
    'id', users_id2  ,'name', users_name2
 ) FROM (
	SELECT
    users2.id as users2_id  ,users2.group_id as users2_group_id  ,users2.id as users_id2  ,users2.name as users_name2
from   users users2
Where users2.id =root.categories1_creator_id
) as root
 )

 )) FROM (
	SELECT
    categories1.id as categories1_id  ,categories1.creator_id as categories1_creator_id  ,categories1.name as categories_name1  ,categories1.id as categories_id1
from   categories categories1 JOIN ticket_category ticket_category1 ON ticket_category1.category_id=categories1.id AND ticket_category1.ticket_id=root.tickets0_id  ORDER BY categories1.created_at desc
) as root
 )
 ,
   'creator', (
   SELECT json_build_object(
    'id', users_id1  ,'name', users_name1
 ) FROM (
	SELECT
    users1.id as users1_id  ,users1.group_id as users1_group_id  ,users1.id as users_id1  ,users1.name as users_name1
from   users users1
Where users1.id =root.tickets0_creator_id
) as root
 )

 ) FROM (
	SELECT
    tickets0.id as tickets0_id  ,tickets0.creator_id as tickets0_creator_id  ,tickets0.tenant_id as tickets0_tenant_id  ,tickets0.name as tickets0_name
from   tickets tickets0  ORDER BY tickets0.created_at desc
) as root