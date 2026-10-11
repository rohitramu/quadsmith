import { useState } from "react";
import { useQuery, useMutation } from "@connectrpc/connect-query";
import { useQueryClient } from "@tanstack/react-query";
import {
  listQuestions,
  listAnswers,
  createQuestion,
  deleteQuestion,
  voteQuestion,
  createAnswer,
  voteAnswer,
  acceptAnswer,
  endorseAnswer,
  createQuestionComment,
  createAnswerComment,
} from "../gen/quadsmith/social-SocialService_connectquery";
import type { Question } from "../gen/quadsmith/social_pb";
import { useAuth } from "../context/AuthContext";
import {
  ChevronUp,
  ChevronDown,
  Check,
  Star,
  Trash2,
  Loader2,
  HelpCircle,
  Plus,
} from "lucide-react";
import { formatRelativeDate } from "../lib/socialFormat";

export interface SocialQaSectionProps {
  target: { type: "build" | "component"; id: string };
  targetAuthorUuid?: string;
  entityName?: string;
}

export function SocialQaSection({ target, targetAuthorUuid, entityName }: SocialQaSectionProps) {
  const { user, isAuthenticated } = useAuth();
  const queryClient = useQueryClient();

  const [sortBy, setSortBy] = useState<string>("votes");
  const [isQuestionComposerOpen, setIsQuestionComposerOpen] = useState(false);
  const [qTitle, setQTitle] = useState("");
  const [qBody, setQBody] = useState("");
  const [composerError, setComposerError] = useState<string | null>(null);
  const [isSubmittingQ, setIsSubmittingQ] = useState(false);

  // Per-question answering state
  const [answeringQuestionId, setAnsweringQuestionId] = useState<string | null>(null);
  const [answerBody, setAnswerBody] = useState("");
  const [isSubmittingAnswer, setIsSubmittingAnswer] = useState(false);

  // Micro comments state
  const [qCommentTexts, setQCommentTexts] = useState<Record<string, string>>({});

  const queryPayload =
    target.type === "build" ? { buildId: target.id, sortBy } : { componentId: target.id, sortBy };

  const {
    data: qResponse,
    isLoading: isLoadingQuestions,
    refetch: refetchQuestions,
  } = useQuery(listQuestions, queryPayload);

  const { mutateAsync: mutateCreateQuestion } = useMutation(createQuestion);
  const { mutateAsync: mutateDeleteQuestion } = useMutation(deleteQuestion);
  const { mutateAsync: mutateVoteQuestion } = useMutation(voteQuestion);
  const { mutateAsync: mutateCreateAnswer } = useMutation(createAnswer);
  const { mutateAsync: mutateCreateQComment } = useMutation(createQuestionComment);

  const questions = qResponse?.questions || [];
  const isBuildAuthor =
    target.type === "build" && !!user?.uuid && !!targetAuthorUuid && user.uuid === targetAuthorUuid;

  async function handleAskQuestion(e: React.FormEvent) {
    e.preventDefault();
    if (!qTitle.trim() || !qBody.trim()) {
      setComposerError("Please provide both a title and description for your question.");
      return;
    }
    setComposerError(null);
    setIsSubmittingQ(true);
    try {
      await mutateCreateQuestion({
        buildId: target.type === "build" ? target.id : undefined,
        componentId: target.type === "component" ? target.id : undefined,
        title: qTitle.trim(),
        body: qBody.trim(),
      });
      setQTitle("");
      setQBody("");
      setIsQuestionComposerOpen(false);
      await refetchQuestions();
      await queryClient.invalidateQueries();
    } catch (err: unknown) {
      setComposerError(err instanceof Error ? err.message : "Failed to post question");
    } finally {
      setIsSubmittingQ(false);
    }
  }

  async function handleVoteQ(questionId: string, currentVote: number, targetVote: number) {
    if (!isAuthenticated) return;
    const newVote = currentVote === targetVote ? 0 : targetVote;
    try {
      await mutateVoteQuestion({ questionId, vote: newVote });
      await refetchQuestions();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to vote");
    }
  }

  async function handlePostAnswer(questionId: string) {
    if (!answerBody.trim()) return;
    setIsSubmittingAnswer(true);
    try {
      await mutateCreateAnswer({
        questionId,
        body: answerBody.trim(),
      });
      setAnswerBody("");
      setAnsweringQuestionId(null);
      await refetchQuestions();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to post answer");
    } finally {
      setIsSubmittingAnswer(false);
    }
  }

  async function handleAddQComment(questionId: string) {
    const text = qCommentTexts[questionId]?.trim();
    if (!text) return;
    try {
      await mutateCreateQComment({ questionId, body: text });
      setQCommentTexts((prev) => ({ ...prev, [questionId]: "" }));
      await refetchQuestions();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to post comment");
    }
  }

  async function handleDeleteQ(questionId: string) {
    if (!window.confirm("Are you sure you want to delete this question?")) return;
    try {
      await mutateDeleteQuestion({ questionId });
      await refetchQuestions();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to delete question");
    }
  }

  return (
    <div className="space-y-6">
      {/* Q&A Header & Ask Question CTA */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-6 rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-zinc-50/70 dark:bg-zinc-900/40">
        <div className="space-y-1">
          <h3 className="font-bold text-base text-zinc-900 dark:text-zinc-100 flex items-center gap-2">
            <HelpCircle size={18} className="text-blue-500" />
            <span>Technical Q&A & Wiring Troubleshooting</span>
          </h3>
          <p className="text-xs text-zinc-500 dark:text-zinc-400">
            Have questions about component fitment, motor KV pairing, or firmware configurations?
            Ask the community and the build creator.
          </p>
        </div>

        <button
          type="button"
          onClick={() => setIsQuestionComposerOpen((prev) => !prev)}
          className="px-5 py-2.5 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs transition-colors shrink-0 shadow-lg shadow-blue-500/20 flex items-center gap-1.5 justify-center cursor-pointer"
        >
          <Plus size={14} />
          <span>{isQuestionComposerOpen ? "Cancel Question" : "Ask a Question"}</span>
        </button>
      </div>

      {/* Ask Question Composer */}
      {isQuestionComposerOpen && (
        <form
          onSubmit={handleAskQuestion}
          className="rounded-2xl border-2 border-blue-500/30 bg-white dark:bg-zinc-900/90 p-6 space-y-4 shadow-xl"
        >
          <div className="flex items-center justify-between">
            <h3 className="font-bold text-sm text-zinc-900 dark:text-zinc-100">
              Ask a Technical Question {entityName ? `about ${entityName}` : ""}
            </h3>
            <button
              type="button"
              onClick={() => setIsQuestionComposerOpen(false)}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 text-xs"
            >
              Cancel
            </button>
          </div>

          {composerError && (
            <div className="p-3 rounded-xl bg-rose-500/10 border border-rose-500/20 text-rose-500 text-xs">
              {composerError}
            </div>
          )}

          <div className="space-y-1.5">
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Question Title
            </label>
            <input
              type="text"
              value={qTitle}
              onChange={(e) => setQTitle(e.target.value)}
              placeholder="e.g., What length standoffs are needed to fit the DJI O3 air unit?"
              className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
            />
          </div>

          <div className="space-y-1.5">
            <label className="block text-xs font-semibold text-zinc-600 dark:text-zinc-400">
              Details & Context
            </label>
            <textarea
              rows={4}
              value={qBody}
              onChange={(e) => setQBody(e.target.value)}
              placeholder="Include your exact hardware variant, symptoms, firmware version, or wiring attempt..."
              className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 p-3 text-sm text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden focus:border-blue-500"
            />
          </div>

          <div className="flex justify-end gap-3 pt-1">
            <button
              type="submit"
              disabled={isSubmittingQ}
              className="px-5 py-2 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-semibold text-xs shadow-md transition-colors flex items-center gap-1.5 disabled:opacity-50"
            >
              {isSubmittingQ && <Loader2 size={13} className="animate-spin" />}
              <span>Submit Question (+5 Rep when upvoted)</span>
            </button>
          </div>
        </form>
      )}

      {/* Questions List & Sort */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <span className="text-xs font-bold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
            Questions & Answers ({questions.length})
          </span>
          <div className="flex items-center gap-2 text-xs">
            <span className="text-zinc-400">Sort by:</span>
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value)}
              className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-2 py-1 text-zinc-700 dark:text-zinc-300 focus:outline-hidden text-xs"
            >
              <option value="votes">Most Votes</option>
              <option value="recent">Most Recent</option>
              <option value="unanswered">Unanswered</option>
            </select>
          </div>
        </div>

        {isLoadingQuestions ? (
          <div className="py-12 flex justify-center text-zinc-400">
            <Loader2 className="animate-spin text-blue-500" size={24} />
          </div>
        ) : questions.length === 0 ? (
          <div className="p-8 text-center rounded-2xl border border-dashed border-zinc-200 dark:border-zinc-800 text-zinc-400 text-xs">
            No technical questions yet. Ask something about this {target.type}!
          </div>
        ) : (
          questions.map((q) => {
            const isAsker = user?.uuid === q.userUuid;
            const comments = q.comments || [];

            return (
              <div
                key={q.id}
                className="rounded-2xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900/60 p-6 space-y-6 shadow-2xs"
              >
                {/* Question Header & Voting Column */}
                <div className="flex items-start gap-4">
                  {/* Voting column */}
                  <div className="flex flex-col items-center gap-0.5 shrink-0 pt-1">
                    <button
                      type="button"
                      onClick={() => handleVoteQ(q.id, q.currentUserVote, 1)}
                      className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                        q.currentUserVote === 1
                          ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                          : "hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-400 hover:text-blue-500"
                      }`}
                      title="Upvote (+5 rep to author)"
                    >
                      <ChevronUp size={20} />
                    </button>
                    <span className="font-bold text-sm font-mono text-zinc-900 dark:text-zinc-100">
                      {q.netScore}
                    </span>
                    <button
                      type="button"
                      onClick={() => handleVoteQ(q.id, q.currentUserVote, -1)}
                      className={`p-1.5 rounded-lg transition-colors cursor-pointer ${
                        q.currentUserVote === -1
                          ? "bg-rose-500/10 text-rose-600 dark:text-rose-400"
                          : "hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-400 hover:text-rose-500"
                      }`}
                      title="Downvote (-2 rep to author, costs you -2 rep)"
                    >
                      <ChevronDown size={20} />
                    </button>
                  </div>

                  {/* Question details */}
                  <div className="space-y-2 flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      {q.hasAcceptedAnswer ? (
                        <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 flex items-center gap-1">
                          <Check size={10} />
                          <span>Solved</span>
                        </span>
                      ) : (
                        <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-zinc-100 dark:bg-zinc-800 text-zinc-500 text-[10px]">
                          Open
                        </span>
                      )}
                      <span className="text-xs text-zinc-400">•</span>
                      <span className="text-xs text-zinc-500 dark:text-zinc-400">
                        Asked by{" "}
                        <span className="font-medium text-zinc-800 dark:text-zinc-200">
                          @{q.authorUsername || "Pilot"}
                        </span>
                      </span>
                      <span className="text-[11px] text-zinc-400">
                        {formatRelativeDate(q.createdAt?.seconds)}
                      </span>
                      {isAsker && (
                        <button
                          type="button"
                          onClick={() => handleDeleteQ(q.id)}
                          className="text-zinc-400 hover:text-rose-500 p-0.5 ml-auto cursor-pointer"
                          title="Delete question"
                        >
                          <Trash2 size={13} />
                        </button>
                      )}
                    </div>

                    <h2 className="text-base font-bold text-zinc-900 dark:text-zinc-100 leading-snug">
                      {q.title}
                    </h2>

                    <div className="text-sm text-zinc-700 dark:text-zinc-300 leading-relaxed whitespace-pre-line">
                      {q.body}
                    </div>

                    {/* Micro comments under question */}
                    {comments.length > 0 && (
                      <div className="pt-2 border-t border-zinc-100 dark:border-zinc-800/80 text-xs text-zinc-600 dark:text-zinc-400 space-y-1.5 pl-2">
                        {comments.map((c) => (
                          <div key={c.uuid} className="flex items-start gap-2">
                            <span className="text-zinc-400 font-mono">↳</span>
                            <div className="flex-1">
                              <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                                @{c.authorUsername}:{" "}
                              </span>
                              <span>{c.body}</span>
                              <span className="text-[10px] text-zinc-400 ml-1.5">
                                • {formatRelativeDate(c.createdAt?.seconds)}
                              </span>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {/* Add comment under question */}
                    {isAuthenticated && (
                      <div className="pt-2 flex items-center gap-2">
                        <input
                          type="text"
                          value={qCommentTexts[q.id] || ""}
                          onChange={(e) =>
                            setQCommentTexts((prev) => ({ ...prev, [q.id]: e.target.value }))
                          }
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              handleAddQComment(q.id);
                            }
                          }}
                          placeholder="Add a clarification comment..."
                          className="flex-1 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 px-3 py-1 text-xs text-zinc-900 dark:text-zinc-200 placeholder-zinc-400 focus:outline-hidden"
                        />
                        <button
                          type="button"
                          onClick={() => handleAddQComment(q.id)}
                          className="px-2.5 py-1 rounded-lg bg-zinc-200 dark:bg-zinc-800 hover:bg-zinc-300 dark:hover:bg-zinc-700 text-xs text-zinc-700 dark:text-zinc-300 font-medium"
                        >
                          Comment
                        </button>
                      </div>
                    )}
                  </div>
                </div>

                {/* Answers Section */}
                <div className="space-y-4 pt-4 border-t border-zinc-200 dark:border-zinc-800">
                  <div className="flex items-center justify-between text-xs font-bold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
                    <span>{q.answerCount} Answers</span>
                    <button
                      type="button"
                      onClick={() =>
                        setAnsweringQuestionId(answeringQuestionId === q.id ? null : q.id)
                      }
                      className="text-blue-500 hover:underline cursor-pointer flex items-center gap-1 font-semibold"
                    >
                      <Plus size={12} />
                      <span>Answer This Question</span>
                    </button>
                  </div>

                  {/* Inline Answer Composer */}
                  {answeringQuestionId === q.id && (
                    <div className="p-4 rounded-xl border border-blue-500/30 bg-blue-500/5 space-y-3">
                      <label className="block text-xs font-semibold text-zinc-700 dark:text-zinc-300">
                        Your Answer
                      </label>
                      <textarea
                        rows={3}
                        value={answerBody}
                        onChange={(e) => setAnswerBody(e.target.value)}
                        placeholder="Provide an actionable solution, exact clearance dimensions, or wiring pinout..."
                        className="w-full rounded-xl border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-950 p-3 text-xs text-zinc-900 dark:text-zinc-100 placeholder-zinc-400 focus:outline-hidden"
                      />
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          onClick={() => setAnsweringQuestionId(null)}
                          className="px-3 py-1.5 rounded-lg text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300"
                        >
                          Cancel
                        </button>
                        <button
                          type="button"
                          disabled={isSubmittingAnswer || !answerBody.trim()}
                          onClick={() => handlePostAnswer(q.id)}
                          className="px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold flex items-center gap-1 shadow-xs disabled:opacity-50"
                        >
                          {isSubmittingAnswer && <Loader2 size={12} className="animate-spin" />}
                          <span>Post Answer (+30 Rep if Accepted)</span>
                        </button>
                      </div>
                    </div>
                  )}

                  {/* Answers List */}
                  <QuestionAnswersList
                    question={q}
                    isBuildAuthor={isBuildAuthor}
                    onAnswerUpdated={() => refetchQuestions()}
                  />
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

function QuestionAnswersList({
  question,
  isBuildAuthor,
  onAnswerUpdated,
}: {
  question: Question;
  isBuildAuthor: boolean;
  onAnswerUpdated: () => void;
}) {
  const { user, isAuthenticated } = useAuth();
  const {
    data: answersRes,
    isLoading,
    refetch,
  } = useQuery(listAnswers, { questionId: question.id });

  const { mutateAsync: mutateVoteAnswer } = useMutation(voteAnswer);
  const { mutateAsync: mutateAcceptAnswer } = useMutation(acceptAnswer);
  const { mutateAsync: mutateEndorseAnswer } = useMutation(endorseAnswer);
  const { mutateAsync: mutateCreateAnsComment } = useMutation(createAnswerComment);

  const [ansCommentTexts, setAnsCommentTexts] = useState<Record<string, string>>({});

  const answers = answersRes?.answers || [];
  const isQuestionAsker = user?.uuid === question.userUuid;

  async function handleVote(answerUuid: string, currentVote: number, targetVote: number) {
    if (!isAuthenticated) return;
    const newVote = currentVote === targetVote ? 0 : targetVote;
    try {
      await mutateVoteAnswer({ answerUuid, vote: newVote });
      await refetch();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to vote on answer");
    }
  }

  async function handleAccept(answerUuid: string) {
    try {
      await mutateAcceptAnswer({ answerUuid });
      await refetch();
      onAnswerUpdated();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to accept answer");
    }
  }

  async function handleEndorse(answerUuid: string) {
    try {
      await mutateEndorseAnswer({ answerUuid });
      await refetch();
      onAnswerUpdated();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to endorse answer");
    }
  }

  async function handleAddComment(answerUuid: string) {
    const text = ansCommentTexts[answerUuid]?.trim();
    if (!text) return;
    try {
      await mutateCreateAnsComment({ answerUuid, body: text });
      setAnsCommentTexts((prev) => ({ ...prev, [answerUuid]: "" }));
      await refetch();
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to post comment");
    }
  }

  if (isLoading) {
    return (
      <div className="py-4 flex justify-center text-zinc-400">
        <Loader2 size={16} className="animate-spin text-blue-500" />
      </div>
    );
  }

  if (answers.length === 0) {
    return <p className="text-xs text-zinc-400 italic py-1">No answers yet for this question.</p>;
  }

  return (
    <div className="space-y-4">
      {answers.map((ans) => {
        const isAccepted = ans.isAcceptedByAsker;
        const isEndorsed = ans.isAuthorEndorsed;
        const comments = ans.comments || [];

        return (
          <div
            key={ans.uuid}
            className={`rounded-xl p-5 space-y-3 transition-colors ${
              isAccepted
                ? "border border-emerald-500/40 bg-emerald-500/5 dark:bg-emerald-950/20"
                : isEndorsed
                  ? "border border-amber-500/40 bg-amber-500/5 dark:bg-amber-950/20"
                  : "border border-zinc-200 dark:border-zinc-800/80 bg-zinc-50/50 dark:bg-zinc-950/40"
            }`}
          >
            {/* Header: Badges & Author info */}
            <div className="flex items-center justify-between flex-wrap gap-2">
              <div className="flex items-center gap-2 flex-wrap">
                {/* Accepted Answer Badge */}
                {isAccepted ? (
                  <div className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-emerald-500 text-zinc-950 font-bold text-xs shadow-xs">
                    <Check size={14} strokeWidth={3} />
                    <span>Accepted Answer</span>
                  </div>
                ) : isQuestionAsker ? (
                  <button
                    type="button"
                    onClick={() => handleAccept(ans.uuid)}
                    className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-dashed border-emerald-500/40 hover:bg-emerald-500/10 text-emerald-600 dark:text-emerald-400 text-xs font-semibold cursor-pointer"
                  >
                    <Check size={13} />
                    <span>Accept as Solution (+30 Rep)</span>
                  </button>
                ) : null}

                {/* Author Verified Star Badge */}
                {isEndorsed ? (
                  <div className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-amber-500/20 border border-amber-500/30 text-amber-600 dark:text-amber-400 font-bold text-xs">
                    <Star size={13} className="fill-current" />
                    <span>Author Verified</span>
                  </div>
                ) : isBuildAuthor ? (
                  <button
                    type="button"
                    onClick={() => handleEndorse(ans.uuid)}
                    className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-dashed border-amber-500/40 hover:bg-amber-500/10 text-amber-600 dark:text-amber-400 text-xs font-semibold cursor-pointer"
                  >
                    <Star size={13} />
                    <span>Verify as Creator (+20 Rep)</span>
                  </button>
                ) : null}
              </div>

              <div className="text-[11px] text-zinc-500 dark:text-zinc-400">
                Answered by{" "}
                <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                  @{ans.authorUsername || "Pilot"}
                </span>{" "}
                • {formatRelativeDate(ans.createdAt?.seconds)}
              </div>
            </div>

            {/* Answer Body with Voting Column */}
            <div className="flex items-start gap-4">
              <div className="flex flex-col items-center gap-0.5 shrink-0 pt-0.5">
                <button
                  type="button"
                  onClick={() => handleVote(ans.uuid, ans.currentUserVote, 1)}
                  className={`p-1 rounded-md transition-colors cursor-pointer ${
                    ans.currentUserVote === 1
                      ? "bg-blue-500/10 text-blue-600 dark:text-blue-400"
                      : "hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-400 hover:text-blue-500"
                  }`}
                  title="Upvote (+5 rep)"
                >
                  <ChevronUp size={16} />
                </button>
                <span className="font-bold text-xs font-mono text-zinc-900 dark:text-zinc-100">
                  {ans.netScore}
                </span>
                <button
                  type="button"
                  onClick={() => handleVote(ans.uuid, ans.currentUserVote, -1)}
                  className={`p-1 rounded-md transition-colors cursor-pointer ${
                    ans.currentUserVote === -1
                      ? "bg-rose-500/10 text-rose-600 dark:text-rose-400"
                      : "hover:bg-zinc-200 dark:hover:bg-zinc-800 text-zinc-400 hover:text-rose-500"
                  }`}
                  title="Downvote (-2 rep)"
                >
                  <ChevronDown size={16} />
                </button>
              </div>

              <div className="text-sm text-zinc-800 dark:text-zinc-200 leading-relaxed whitespace-pre-line flex-1">
                {ans.body}
              </div>
            </div>

            {/* Micro comments under answer */}
            {comments.length > 0 && (
              <div className="pt-2 border-t border-zinc-200 dark:border-zinc-800/80 text-xs text-zinc-600 dark:text-zinc-400 pl-4 space-y-1">
                {comments.map((c) => (
                  <p key={c.uuid}>
                    <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                      @{c.authorUsername}:{" "}
                    </span>
                    <span>{c.body}</span>
                    <span className="text-[10px] text-zinc-400 ml-1.5">
                      • {formatRelativeDate(c.createdAt?.seconds)}
                    </span>
                  </p>
                ))}
              </div>
            )}

            {/* Add comment under answer */}
            {isAuthenticated && (
              <div className="pt-1 pl-4 flex items-center gap-2">
                <input
                  type="text"
                  value={ansCommentTexts[ans.uuid] || ""}
                  onChange={(e) =>
                    setAnsCommentTexts((prev) => ({ ...prev, [ans.uuid]: e.target.value }))
                  }
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      e.preventDefault();
                      handleAddComment(ans.uuid);
                    }
                  }}
                  placeholder="Clarify this answer..."
                  className="flex-1 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-950 px-2.5 py-1 text-xs text-zinc-900 dark:text-zinc-200 placeholder-zinc-400 focus:outline-hidden"
                />
                <button
                  type="button"
                  onClick={() => handleAddComment(ans.uuid)}
                  className="px-2.5 py-1 rounded-lg bg-zinc-200 dark:bg-zinc-800 hover:bg-zinc-300 dark:hover:bg-zinc-700 text-xs text-zinc-700 dark:text-zinc-300 font-medium"
                >
                  Reply
                </button>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
