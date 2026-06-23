import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from '../auth/auth.service';
import { AuthzService } from './authz.service';

export const permissionGuard: CanActivateFn = (route, state) => {
  const auth = inject(AuthService);
  const authz = inject(AuthzService);
  const router = inject(Router);
  const requiredPermission = route.data['permission'] as string | undefined;
  const requiredPermissions = route.data['permissions'] as string[] | undefined;
  const permissionsMode = (route.data['permissionsMode'] as 'all' | 'any' | undefined) ?? 'all';
  const requiredRole = route.data['role'] as string | undefined;
  const requiredRoles = route.data['roles'] as string[] | undefined;
  const rolesMode = (route.data['rolesMode'] as 'all' | 'any' | undefined) ?? 'any';
  const requiredModule = route.data['module'] as string | undefined;
  const requiredModules = route.data['modules'] as string[] | undefined;
  const modulesMode = (route.data['modulesMode'] as 'all' | 'any' | undefined) ?? 'all';

  if (!authz.user()) {
    auth.storeReturnUrl(state.url);
    return router.createUrlTree(['/']);
  }

  if (authz.hasFullAccess()) {
    return true;
  }

  const allowed =
    (!requiredRole || authz.hasRole(requiredRole)) &&
    (!requiredRoles ||
      (rolesMode === 'any'
        ? authz.hasAnyRole(requiredRoles)
        : requiredRoles.every((role) => authz.hasRole(role)))) &&
    (!requiredPermission || authz.hasPermission(requiredPermission)) &&
    (!requiredPermissions ||
      (permissionsMode === 'any'
        ? authz.hasAnyPermission(requiredPermissions)
        : requiredPermissions.every((permission) => authz.hasPermission(permission)))) &&
    (!requiredModule || authz.hasModule(requiredModule)) &&
    (!requiredModules ||
      (modulesMode === 'any'
        ? authz.hasAnyModule(requiredModules)
        : requiredModules.every((moduleCode) => authz.hasModule(moduleCode))));

  return allowed
    ? true
    : router.createUrlTree(['/auth/access-denied'], { queryParams: { from: state.url } });
};
