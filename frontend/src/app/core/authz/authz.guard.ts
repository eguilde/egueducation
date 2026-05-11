import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthzService } from './authz.service';

export const permissionGuard: CanActivateFn = (route) => {
  const authz = inject(AuthzService);
  const router = inject(Router);
  const requiredPermission = route.data['permission'] as string | undefined;
  const requiredModule = route.data['module'] as string | undefined;

  const allowed =
    (!requiredPermission || authz.hasPermission(requiredPermission)) &&
    (!requiredModule || authz.hasModule(requiredModule));

  return allowed ? true : router.createUrlTree(['/dashboard']);
};
