
import { useState } from "react";

import { Button } from "@components/Button";
import { InputFieldWithLabel } from "@/components/InputField";
import { NetworkState } from "@/hooks/stores";



export interface StaticIPConfig {
  address: string;
  netmask: string;
  gateway: string;
  dns: string[];
}

export default function StaticIpCard({
  networkState,
  setStaticIPConfig
}: {
  networkState: NetworkState;
  setStaticIPConfig: (value: StaticIPConfig) => void;

}) {
  

const [staticIPConfig, setipv4staticState] = useState<StaticIPConfig>({
  address: "",
  netmask: "",
  gateway: "",
  dns: []
});


const handleAddressChange = (value: string) => {
  setipv4staticState({ ...staticIPConfig, address: value });
};

const handleNetmaskChange = (value: string) => {
  setipv4staticState({ ...staticIPConfig, netmask: value });
};

const handleGatewayChange = (value: string) => {
  setipv4staticState({ ...staticIPConfig, gateway: value });
};

const handleDNSChange = (value: string) => {
  setipv4staticState({ ...staticIPConfig, dns: [value] });
};

const _setNetworkSettings = () => {
  setStaticIPConfig(staticIPConfig)
};


console.log(networkState)

  return (
    <div className="">
      <div className="grid grid-cols-2 gap-4">
        <InputFieldWithLabel
          required
          label="Address"
          placeholder="Enter Address"
          defaultValue={staticIPConfig?.address}
          onChange={e => handleAddressChange(e.target.value)}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <InputFieldWithLabel
          required
          label="Netmask"
          placeholder="Enter Netmask"
          defaultValue={staticIPConfig?.netmask}
          onChange={e => handleNetmaskChange(e.target.value)}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <InputFieldWithLabel
          required
          label="Gateway"
          placeholder="Enter Gateway"
          defaultValue={staticIPConfig?.gateway}
          onChange={e => handleGatewayChange(e.target.value)}
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <InputFieldWithLabel
          required
          label="DNS"
          placeholder="Enter DNS"
          defaultValue={staticIPConfig?.dns[0]}
          onChange={e => handleDNSChange(e.target.value)}
        />
      </div>
      <div className="mt-6 flex gap-x-2">
        <Button
          loading={false}
          size="SM"
          theme="primary"
          text="Update USB Identifiers"
          onClick={() => _setNetworkSettings()}
        />
      </div>
    </div>
  );
}
