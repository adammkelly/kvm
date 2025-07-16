
import { GridCard } from "@/components/Card";
import { InputFieldWithLabel } from "@/components/InputField";
import Fieldset from "@/components/Fieldset";
import { NetworkState } from "@/hooks/stores";

const ipv4_static = {
  address: "",
  netmask: "",
  gateway: "",
  dns: []
}

export default function StaticIpCard({
  networkState
}: {
  networkState: NetworkState;
}) {
  return (
    <>
    
    {console.log(networkState)}
    <GridCard>
      <div className="animate-fadeIn p-4 opacity-0 animation-duration-500 text-black dark:text-white">
        <div className="space-y-3">
          <h3 className="text-base font-bold text-slate-900 dark:text-white">
            Static IP Information
          </h3>

          <div className="flex gap-x-6 gap-y-2">
            <div className="space-y-4">
              <Fieldset>
                <InputFieldWithLabel
                  type="text"
                  label="Macro Name"
                  placeholder="Macro Name"
                  value="test"
                  onChange={e => { ipv4_static.address = e.target.value}}
                />
              </Fieldset>
            </div>
            <div className="space-y-4">
              <Fieldset>
                <InputFieldWithLabel
                  type="text"
                  label="Macro Name"
                  placeholder="Macro Name"
                  value="test"
                  onChange={e => { ipv4_static.netmask = e.target.value}}
                />
              </Fieldset>
            </div>
            <div className="space-y-4">
              <Fieldset>
                <InputFieldWithLabel
                  type="text"
                  label="Macro Name"
                  placeholder="Macro Name"
                  value="test"
                  onChange={e => { ipv4_static.gateway = e.target.value}}
                />
              </Fieldset>
            </div>
          </div>
        </div>
      </div>
    </GridCard>
    </>
  );
}
