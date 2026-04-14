import { useMutation } from "@apollo/client";
import { useNavigate } from "react-router-dom";
import toast from "react-hot-toast";
import { CREATE_DEBATE, START_DEBATE } from "../graphql/mutations";
import type { CreateDebateInput, Debate } from "../types/debate";

interface CreateDebateData {
  createDebate: Debate;
}

interface StartDebateData {
  startDebate: Debate;
}

export function useCreateDebate() {
  const navigate = useNavigate();
  const [createDebateMutation, createResult] = useMutation<CreateDebateData>(
    CREATE_DEBATE,
  );
  const [startDebateMutation] = useMutation<StartDebateData>(START_DEBATE);

  const createAndStart = async (
    input: CreateDebateInput,
    autoStart = false,
  ): Promise<string | null> => {
    try {
      const { statementsPerRound: _ignored, ...backendInput } = input;
      const created = await createDebateMutation({ variables: { input: backendInput } });
      const debateId = created.data?.createDebate.id;
      if (!debateId) {
        toast.error("Unable to create debate.");
        return null;
      }

      if (autoStart) {
        await startDebateMutation({ variables: { id: debateId } });
      }

      navigate(`/debate/${debateId}`);
      toast.success("Debate created successfully.");
      return debateId;
    } catch (error) {
      const message =
        error instanceof Error ? error.message : "Failed to create debate.";
      toast.error(message);
      return null;
    }
  };

  return {
    createAndStart,
    loading: createResult.loading,
    error: createResult.error,
  };
}
