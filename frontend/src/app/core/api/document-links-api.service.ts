import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import { CreateDocumentLinkRequest, DocumentLookupItem, LinkedDocument } from './api.types';

@Injectable({ providedIn: 'root' })
export class DocumentLinksApiService {
  private readonly http = inject(HttpClient);

  lookupDocuments(query: string) {
    const params = new HttpParams().set('query', query);
    return this.http.get<DocumentLookupItem[]>('/api/registratura/documents/lookup', { params });
  }

  listLinks(sourceModule: string, sourceRecordId: string) {
    const params = new HttpParams()
      .set('source_module', sourceModule)
      .set('source_record_id', sourceRecordId);
    return this.http.get<LinkedDocument[]>('/api/registratura/document-links', { params });
  }

  createLink(payload: CreateDocumentLinkRequest) {
    return this.http.post<LinkedDocument>('/api/registratura/document-links', payload);
  }

  deleteLink(linkId: string) {
    return this.http.delete<void>(`/api/registratura/document-links/${linkId}`);
  }
}
