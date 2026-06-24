import { ChangeDetectionStrategy, Component, computed, inject } from '@angular/core';
import { RouterLink } from '@angular/router';

import { ButtonModule } from 'primeng/button';
import { CardModule } from 'primeng/card';
import { DividerModule } from 'primeng/divider';

import { AuthService } from '../../core/auth/auth.service';
import { AuthzService } from '../../core/authz/authz.service';
import { ThemeService } from '../../core/ui/theme.service';

interface HomeModuleCard {
  title: string;
  description: string;
  icon: string;
  route: string;
  accent: string;
}

@Component({
  selector: 'app-home-page',
  standalone: true,
  imports: [ButtonModule, CardModule, DividerModule, RouterLink],
  template: `
    <section class="home-shell">
      <p-card styleClass="home-shell__hero">
        <div class="home-shell__hero-grid">
          <div class="home-shell__hero-copy">
            <p class="home-shell__eyebrow">Platformă educațională și registratură unificată</p>
            <h1 class="home-shell__title">Gestionare completă pentru educație, documente și arhivă electronică</h1>
            <p class="home-shell__lead">
              eGuEducation oferă fluxuri clare pentru registratură, evidență documente, arhivă electronică,
              RBAC și modulele educaționale cerute de reglementările curente.
            </p>

            <div class="home-shell__actions">
              <a pButton routerLink="/documente" label="Deschide registratura" icon="pi pi-folder-open"></a>
              <a pButton routerLink="/education/dashboard" label="Vezi educația" icon="pi pi-graduation-cap" severity="secondary"></a>
              <a pButton routerLink="/profile" label="Profil" icon="pi pi-user" severity="secondary"></a>
            </div>
          </div>

          <div class="home-shell__status-panel">
            <div class="home-shell__status-badge">Tema curentă</div>
            <div class="home-shell__status-value">
              {{ themeLabel() }}
            </div>
            <p class="home-shell__status-note">
              Paletă red/rose, fundaluri pe baza tokenurilor PrimeNG și mod întunecat automat.
            </p>
            <p-divider />
            <div class="home-shell__status-row">
              <span>Autentificat</span>
              <strong>{{ auth.isAuthenticated() ? 'Da' : 'Nu' }}</strong>
            </div>
            <div class="home-shell__status-row">
              <span>Roluri active</span>
              <strong>{{ authz.roles().length }}</strong>
            </div>
          </div>
        </div>
      </p-card>

      <div class="home-shell__modules">
        @for (item of moduleCards; track item.route) {
          <p-card styleClass="home-shell__module-card">
            <div class="home-shell__module-header">
              <span class="home-shell__module-icon" [style.background]="item.accent">
                <i [class]="item.icon"></i>
              </span>
              <div class="home-shell__module-copy">
                <h2>{{ item.title }}</h2>
                <p>{{ item.description }}</p>
              </div>
            </div>
            <div class="home-shell__module-footer">
              <a pButton [routerLink]="item.route" label="Accesează" icon="pi pi-arrow-right" severity="secondary"></a>
            </div>
          </p-card>
        }
      </div>
    </section>
  `,
  styleUrl: './home-page.component.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class HomePageComponent {
  protected readonly auth = inject(AuthService);
  protected readonly authz = inject(AuthzService);
  protected readonly theme = inject(ThemeService);

  protected readonly moduleCards: HomeModuleCard[] = [
    {
      title: 'Registratură',
      description: 'Documente, numere de înregistrare, trasee și stări operaționale.',
      icon: 'pi pi-folder-open',
      route: '/documente',
      accent: 'var(--p-primary-500)',
    },
    {
      title: 'Educație',
      description: 'Guvernanță, personal, portofolii și conformitate educațională.',
      icon: 'pi pi-graduation-cap',
      route: '/education/dashboard',
      accent: 'var(--p-primary-600)',
    },
    {
      title: 'Arhivă',
      description: 'Depozitare electronică, consultare și urmărire arhivistică.',
      icon: 'pi pi-inbox',
      route: '/earchiva',
      accent: 'var(--p-primary-700)',
    },
    {
      title: 'Profil și RBAC',
      description: 'Date de cont, metode de autentificare și administrare roluri.',
      icon: 'pi pi-user',
      route: '/profile',
      accent: 'var(--p-primary-800)',
    },
  ];

  protected readonly themeLabel = computed(() => (this.theme.isDarkMode() ? 'Red / Dark' : 'Rose / Light'));
}
