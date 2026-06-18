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
