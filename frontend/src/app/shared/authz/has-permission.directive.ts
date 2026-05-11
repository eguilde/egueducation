import {
  Directive,
  TemplateRef,
  ViewContainerRef,
  computed,
  effect,
  inject,
  input,
} from '@angular/core';

import { AuthzService } from '../../core/authz/authz.service';

@Directive({
  selector: '[appHasPermission]',
  standalone: true,
})
export class HasPermissionDirective {
  private readonly templateRef = inject(TemplateRef<unknown>);
  private readonly viewContainer = inject(ViewContainerRef);
  private readonly authz = inject(AuthzService);

  readonly appHasPermission = input<string | string[] | undefined>(undefined);
  readonly appHasPermissionModule = input<string | undefined>(undefined);
  readonly appHasPermissionMode = input<'all' | 'any'>('all');

  private readonly allowed = computed(() => {
    const required = this.appHasPermission();
    const moduleCode = this.appHasPermissionModule();
    const mode = this.appHasPermissionMode();

    const permissionOk =
      !required ||
      (Array.isArray(required)
        ? mode === 'any'
          ? this.authz.hasAnyPermission(required)
          : required.every((permission) => this.authz.hasPermission(permission))
        : this.authz.hasPermission(required));
    const moduleOk = !moduleCode || this.authz.hasModule(moduleCode);

    return permissionOk && moduleOk;
  });

  constructor() {
    effect(() => {
      if (this.allowed()) {
        if (this.viewContainer.length === 0) {
          this.viewContainer.createEmbeddedView(this.templateRef);
        }
        return;
      }

      this.viewContainer.clear();
    });
  }
}
