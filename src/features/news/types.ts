export interface NewsItem {
  id: string;
  teamId: string;
  title: string;
  body: string;
  authorId: string;
  pinned: boolean;
  createdAt: string;
  authorName?: string;
  authorColor?: string;
  authorPhoto?: string | null;
}

/** Editing buffer shape for the news create/edit sheet. `id` is set when editing. */
export interface NewsFormValues extends Record<string, unknown> {
  id?: string;
  title: string;
  body: string;
  pinned: boolean;
}
