import { describe, it, expect, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test/test-utils";
import { SocialToolbar } from "../SocialToolbar";
import { SaveToCollectionModal } from "../SaveToCollectionModal";
import { SocialReviewSection } from "../SocialReviewSection";
import { SocialQaSection } from "../SocialQaSection";
import { SocialTabs } from "../SocialTabs";

describe("Social Interactions Components", () => {
  const target = { type: "build" as const, id: "bando-basher-5-inch" };
  const targetName = "Bando Basher 5 inch";

  describe("SocialToolbar", () => {
    it("renders like count, sentiment pill, and save button", async () => {
      const { user } = renderWithProviders(
        <SocialToolbar target={target} targetName={targetName} />,
      );

      // Verify like button renders with count 42
      const likeBtn = await screen.findByTitle("Unlike");
      expect(likeBtn).toBeInTheDocument();
      expect(screen.getByText("42")).toBeInTheDocument();

      // Verify sentiment pill
      expect(screen.getByText("Very Positive")).toBeInTheDocument();
      expect(screen.getByText("92%")).toBeInTheDocument();

      // Verify Q&A count
      expect(screen.getByText(/Q&A \(3\)/i)).toBeInTheDocument();

      // Verify Save button
      const saveBtn = screen.getByRole("button", { name: /save to collection/i });
      expect(saveBtn).toBeInTheDocument();

      // Click like button to toggle like
      await user.click(likeBtn);
      // Count toggles to 41
      await waitFor(() => {
        expect(screen.getByText("41")).toBeInTheDocument();
      });
    });

    it("opens save to collection modal when save button is clicked", async () => {
      const { user } = renderWithProviders(
        <SocialToolbar target={target} targetName={targetName} />,
      );

      const saveBtn = await screen.findByRole("button", { name: /save to collection/i });
      await user.click(saveBtn);

      // Modal appears
      expect(await screen.findByRole("dialog")).toBeInTheDocument();
      expect(screen.getByRole("heading", { name: /save to collection/i })).toBeInTheDocument();
    });
  });

  describe("SaveToCollectionModal", () => {
    it("lists user collections, indicates added items with solid circle tick icon, and allows toggling membership", async () => {
      const onClose = vi.fn();
      const { user } = renderWithProviders(
        <SaveToCollectionModal
          isOpen={true}
          onClose={onClose}
          target={target}
          targetName={targetName}
        />,
      );

      // Should display modal title
      expect(screen.getByRole("heading", { name: /save to collection/i })).toBeInTheDocument();

      // System collection "Liked" must NOT be displayed in Save to Collection modal
      expect(screen.queryByRole("button", { name: /liked/i })).not.toBeInTheDocument();

      // Should list user's custom collections
      const cinewhoopBtn = await screen.findByRole("button", {
        name: /remove.*from cinewhoop favorites/i,
      });
      const bandoBtn = screen.getByRole("button", {
        name: /add.*to bando builds/i,
      });

      expect(cinewhoopBtn).toBeInTheDocument();
      expect(bandoBtn).toBeInTheDocument();

      // Cinewhoop Favorites already contains target: should have tick icon inside solid circle and Added badge
      expect(cinewhoopBtn.querySelector('[data-testid="tick-icon"]')).toBeInTheDocument();
      expect(screen.getByText("Added")).toBeInTheDocument();

      // Bando Builds does not yet contain target: should have plus icon
      expect(bandoBtn.querySelector('[data-testid="plus-icon"]')).toBeInTheDocument();

      // Click "Bando Builds" to add build to collection: plus icon changes to tick icon
      await user.click(bandoBtn);
      expect(bandoBtn.querySelector('[data-testid="tick-icon"]')).toBeInTheDocument();

      // Click "Cinewhoop Favorites" to remove build: tick icon changes to plus icon
      await user.click(cinewhoopBtn);
      expect(cinewhoopBtn.querySelector('[data-testid="plus-icon"]')).toBeInTheDocument();

      // Public collection checkbox is removed from the quick create form
      expect(screen.queryByLabelText(/public collection/i)).not.toBeInTheDocument();

      // Create new collection
      const input = screen.getByPlaceholderText(/Collection name/i);
      await user.type(input, "Racing Rigs");

      const createBtn = screen.getByRole("button", { name: /create/i });
      await user.click(createBtn);

      // Newly created collection should appear with tick icon because item was auto-added
      const racingRigsBtn = await screen.findByRole("button", {
        name: /remove.*from racing rigs/i,
      });
      expect(racingRigsBtn).toBeInTheDocument();
      expect(racingRigsBtn.querySelector('[data-testid="tick-icon"]')).toBeInTheDocument();
      expect(racingRigsBtn).toHaveTextContent(/1\s*items/i);
    });
  });

  describe("SocialReviewSection", () => {
    it("displays reviews summary, review cards, and allows posting a review", async () => {
      const { user } = renderWithProviders(
        <SocialReviewSection target={target} entityName={targetName} />,
      );

      // Steam-style consensus tier (appears in badge and progress description)
      const tierBadges = await screen.findAllByText(/Very Positive/i);
      expect(tierBadges.length).toBeGreaterThan(0);
      expect(screen.getByText(/92% Recommended/i)).toBeInTheDocument();

      // Review card author and body
      expect(await screen.findByText(/rotor_king/i)).toBeInTheDocument();
      expect(screen.getByText(/Incredible frame stiffness/i)).toBeInTheDocument();
      expect(screen.getByText(/Verified Owner/i)).toBeInTheDocument();

      // Click "Write a Review"
      const writeReviewBtn = screen.getByRole("button", { name: /write a review/i });
      await user.click(writeReviewBtn);

      // Form appears
      const textarea = screen.getByPlaceholderText(/Describe flight characteristics/i);
      expect(textarea).toBeInTheDocument();

      await user.type(textarea, "Flies like a dream! Super rigid arms.");
      const submitBtn = screen.getByRole("button", { name: /post review/i });
      await user.click(submitBtn);

      // The new review should be listed
      expect(await screen.findByText(/Flies like a dream! Super rigid arms./i)).toBeInTheDocument();
    });

    it("allows voting a review as helpful", async () => {
      const { user } = renderWithProviders(
        <SocialReviewSection target={target} entityName={targetName} />,
      );

      // Helpful button with initial count 5
      const helpfulBtn = await screen.findByRole("button", { name: /yes\s*5/i });
      expect(helpfulBtn).toBeInTheDocument();

      await user.click(helpfulBtn);

      // Count increases to 6 and displays directional label
      expect(await screen.findByRole("button", { name: /you voted yes\s*6/i })).toBeInTheDocument();
    });
  });

  describe("SocialQaSection", () => {
    it("renders questions with net score and answers with badges", async () => {
      renderWithProviders(<SocialQaSection target={target} entityName={targetName} />);

      // Question title and score
      expect(
        await screen.findByText(/Does this frame fit 20x20mm mounting stacks\?/i),
      ).toBeInTheDocument();
      expect(screen.getByText("3")).toBeInTheDocument(); // net score
      expect(screen.getByText(/Solved/i)).toBeInTheDocument();

      // Answers should be displayed
      expect(await screen.findByText(/Yes, you can use 20x20mm stacks/i)).toBeInTheDocument();

      // Endorsement badges
      expect(screen.getByText(/Accepted Answer/i)).toBeInTheDocument();
      expect(screen.getByText(/Author Verified/i)).toBeInTheDocument();
    });

    it("allows opening question composer and submitting a question", async () => {
      const { user } = renderWithProviders(
        <SocialQaSection target={target} entityName={targetName} />,
      );

      const askBtn = await screen.findByRole("button", { name: /ask a question/i });
      await user.click(askBtn);

      const titleInput = screen.getByPlaceholderText(/What length standoffs are needed/i);
      const bodyInput = screen.getByPlaceholderText(/Include your exact hardware variant/i);

      await user.type(titleInput, "What battery strap length is recommended?");
      await user.type(bodyInput, "Is 250mm long enough for a 6S 1400mAh pack?");

      const postBtn = screen.getByRole("button", { name: /submit question/i });
      await user.click(postBtn);

      // Newly posted question appears
      expect(
        await screen.findByText("What battery strap length is recommended?"),
      ).toBeInTheDocument();
    });

    it("prevents build author from asking questions on their own build", async () => {
      renderWithProviders(
        <SocialQaSection
          target={target}
          targetAuthorUuid="018f0000-0000-7000-0000-000000000099" // mockUser.uuid
          entityName={targetName}
        />,
      );

      expect(
        await screen.findByText(/Build creators cannot ask questions on their own builds/i),
      ).toBeInTheDocument();
      expect(screen.queryByRole("button", { name: /ask a question/i })).not.toBeInTheDocument();
    });
  });

  describe("SocialTabs", () => {
    it("allows switching tabs and viewing community reputation rules", async () => {
      const { user } = renderWithProviders(<SocialTabs target={target} entityName={targetName} />);

      // Initial active tab is User Reviews
      expect(await screen.findByText(/Community Sentiment/i)).toBeInTheDocument();

      // Switch to Q&A tab
      const qaTab = screen.getByRole("button", { name: /technical q&a/i });
      await user.click(qaTab);
      expect(
        await screen.findByRole("heading", { level: 3, name: /^technical q&a$/i }),
      ).toBeInTheDocument();

      // Switch to Collections tab
      const collectionsTab = screen.getByRole("button", { name: /collections/i });
      await user.click(collectionsTab);
      expect(await screen.findByText("Cinewhoop Favorites")).toBeInTheDocument();
      expect(screen.queryByText("Liked")).not.toBeInTheDocument();

      // Verify reputation economy content card
      expect(screen.getByText(/Community Reputation Economy \(Karma Rules\)/i)).toBeInTheDocument();
      expect(screen.getByText(/\+30 Rep/i)).toBeInTheDocument();
      expect(screen.getAllByText(/Accepted Answer/i).length).toBeGreaterThan(0);
      expect(screen.getByText(/\+20 Rep/i)).toBeInTheDocument();
      expect(screen.getByText(/\+3 Rep/i)).toBeInTheDocument();
      expect(screen.getByText(/Review Author/i)).toBeInTheDocument();
      expect(screen.getByText(/\+1 Rep/i)).toBeInTheDocument();
      expect(screen.getByText(/-2 Rep \(Both\)/i)).toBeInTheDocument();
      expect(screen.getByText(/Unlock at 100 rep/i)).toBeInTheDocument();
    });
  });
});
