// page.tsx
// 学校管理员成绩审核详情页。

import { GradePanelsAdminWrapper } from "@/components/business/GradePanelsWrapper";

/** AdminGradesReviewDetailPage 学校管理员成绩审核详情页面。 */
export default function AdminGradesReviewDetailPage({ params }: { params: { id: string } }) {
  return <GradePanelsAdminWrapper page="review-detail" reviewID={params.id} />;
}
