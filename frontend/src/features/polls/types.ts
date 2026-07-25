export interface PollVoter {
  /** Stable user id — matrix rows key off this. Empty string only for the
   *  anonymous case, which never produces voter entries anyway. */
  userId: string;
  /** Membership id in the poll's team, used to build the photo URL. `null`
   *  when the voter is no longer a team member (vote survives, photo can't). */
  membershipId: string | null;
  name: string;
  color: string;
  photo: string | null;
}

export interface PollOption {
  id: string;
  text: string;
  count: number;
  pct: number;
  voters: PollVoter[];
}

export interface Poll {
  id: string;
  question: string;
  multiple: boolean;
  anonymous: boolean;
  createdAt: string;
  totalVotes: number;
  myVote: string[] | null;
  options: PollOption[];
}

/** Raw poll option as stored server-side (no computed vote counts/percentages). */
export interface PollOptionDto {
  id: string;
  text: string;
}

/** A single member's vote on a poll (one entry per user; `optionIds` has >1 entry only for `multiple` polls). */
export interface PollVoteDto {
  userId: string;
  optionIds: string[];
}

/** Raw poll payload as it should be returned/stored by polls.* API endpoints, before vote-count enrichment. */
export interface PollDto {
  id: string;
  teamId: string;
  question: string;
  multiple: boolean;
  anonymous: boolean;
  createdAt: string;
  options: PollOptionDto[];
  votes: PollVoteDto[];
}

/** Editing buffer shape for the poll-creation sheet. */
export interface PollFormValues extends Record<string, unknown> {
  question: string;
  opt0: string;
  opt1: string;
  opt2: string;
  opt3: string;
  multiple: boolean;
  anonymous: boolean;
}
