# Copyright AGNTCY Contributors (https://github.com/agntcy)
# SPDX-License-Identifier: Apache-2.0

from llama_index.core.workflow import Workflow, StartEvent, StopEvent, step
from random import randint

class NumGenWorkflow(Workflow):
    @step()
    async def run_step(self, ev: StartEvent) -> StopEvent:
        max_val = ev.get("max")
        if not max_val:
            raise ValueError("max_val is required.")
        num = randint(1, max_val)
        print("return " + str(num))
        return StopEvent(result=num)


class SumWorkflow(Workflow):
    @step()
    async def run_step(
        self, ev: StartEvent, gen: NumGenWorkflow
    ) -> StopEvent:
        max_val = ev.get("max")
        if not max_val:
            raise ValueError("max is required.")

        num1 = await gen.run(max=max_val)
        num2 = await gen.run(max=max_val)
        return StopEvent(result=(num1 + " + " + num2 + " = " + str(int(num1)+int(num2))))