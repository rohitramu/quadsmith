import { CollectionTableView } from "./CollectionPage";
import { BUILD_COLLECTION } from "../lib/buildCollection";

export function BuildsPage() {
  return <CollectionTableView categoryId="builds" collection={BUILD_COLLECTION} />;
}

export default BuildsPage;
