-- Keep the School Classes FORCE-RLS boundary in lockstep with the HTTP
-- authorization boundary. Institution-scoped, accepted delegations are
-- operational grants for director adjuncts; they must not pass routing and
-- then be denied by an otherwise direct-only table policy.
create or replace function public.education_classes_actor_has_permission(p_permission text)
returns boolean language sql stable security definer set search_path = pg_catalog, public as $$
	with actor as (
		select u.id
		from public.app_users u
		where lower(u.sub) = lower(nullif(btrim(current_setting('app.actor_subject', true)), ''))
			and u.status = 'active'
		limit 1
	), current_scope as (
		select public.current_tenant_code() tenant_code, public.current_institution_id() institution_id
	)
	select exists (
		select 1 from actor, current_scope scope
		where scope.tenant_code is not null
			and scope.institution_id is not null
			and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, null)
			and (
				exists (
					select 1 from public.app_user_permissions up
					where up.user_id = actor.id and up.tenant_code = scope.tenant_code and up.permission_code = p_permission
					union all
					select 1 from public.app_user_roles ur
					join public.app_role_permissions rp on rp.role_code = ur.role_code
					where ur.user_id = actor.id and ur.tenant_code = scope.tenant_code and rp.permission_code = p_permission
					union all
					select 1 from public.app_memberships membership
					join public.app_position_permissions pp on pp.position_code = membership.position_code
					where membership.user_id = actor.id and membership.tenant_code = scope.tenant_code and pp.permission_code = p_permission
						and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, membership.position_code)
					union all
					select 1 from public.app_memberships membership
					join public.app_position_roles pr on pr.position_code = membership.position_code
					join public.app_role_permissions rp on rp.role_code = pr.role_code
					where membership.user_id = actor.id and membership.tenant_code = scope.tenant_code and rp.permission_code = p_permission
						and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, membership.position_code)
				)
				or exists (
					select 1
					from public.education_role_delegations delegation
					where delegation.tenant_code = scope.tenant_code
						and delegation.institution_id = scope.institution_id
						and delegation.delegate_user_id = actor.id
						and delegation.permission_code = p_permission
						and delegation.resource_type = 'institution'
						and delegation.resource_id is null
						and delegation.status = 'accepted'
						and delegation.valid_from <= current_date
						and (delegation.valid_until is null or delegation.valid_until >= current_date)
						and public.education_membership_is_eligible(actor.id, scope.tenant_code, scope.institution_id, 'director_adjunct')
						and public.education_membership_is_eligible(delegation.delegator_user_id, scope.tenant_code, scope.institution_id, 'director')
						and exists (
							select 1 from public.app_user_permissions up
							where up.user_id = delegation.delegator_user_id and up.tenant_code = scope.tenant_code and up.permission_code = delegation.permission_code
							union all
							select 1 from public.app_user_roles ur
							join public.app_role_permissions rp on rp.role_code = ur.role_code
							where ur.user_id = delegation.delegator_user_id and ur.tenant_code = scope.tenant_code and rp.permission_code = delegation.permission_code
							union all
							select 1 from public.app_memberships membership
							join public.app_position_permissions pp on pp.position_code = membership.position_code
							where membership.user_id = delegation.delegator_user_id and membership.tenant_code = scope.tenant_code and pp.permission_code = delegation.permission_code
								and public.education_membership_is_eligible(delegation.delegator_user_id, scope.tenant_code, scope.institution_id, membership.position_code)
							union all
							select 1 from public.app_memberships membership
							join public.app_position_roles pr on pr.position_code = membership.position_code
							join public.app_role_permissions rp on rp.role_code = pr.role_code
							where membership.user_id = delegation.delegator_user_id and membership.tenant_code = scope.tenant_code and rp.permission_code = delegation.permission_code
								and public.education_membership_is_eligible(delegation.delegator_user_id, scope.tenant_code, scope.institution_id, membership.position_code)
						)
				)
			)
	);
$$;
revoke all on function public.education_classes_actor_has_permission(text) from public;
grant execute on function public.education_classes_actor_has_permission(text) to public;
