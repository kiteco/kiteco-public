const OCTOBAT_PUBLISHABLE_KEY: string = process.env.REACT_APP_OCTOBAT_PUBLISHABLE_KEY as string;

const headers: Headers = new Headers();
headers.append("Authorization", `Basic ${btoa(OCTOBAT_PUBLISHABLE_KEY)}:`);
headers.append("Content-Type", "application/json");

interface IServerTaxEvidenceRequests {
  customer_billing_address_country: string;
}

interface IResponseTaxEvidenceRequests {
  id: string,
  tax: string,
  tax_details: IResponseTaxDetails[] | null;
}

interface IResponseTaxDetails {
  rate: number,
  tax: string,
  tax_name: string,
  name: string,
}

export interface ITaxDetails {
  id: string,
  tax?: string,
  name?: string,
  // this is a percentage in [0,100], not [0,1]
  rate?: number,
  jurisdiction?: string,
}

export async function retrieveTaxDetails(countryISOCode: string): Promise<ITaxDetails | null> {
  const body: IServerTaxEvidenceRequests = {
    customer_billing_address_country: countryISOCode,
  }

  const response: IResponseTaxEvidenceRequests = await fetch(
    `https://apiv2.octobat.com/tax_evidence_requests`,
    {
      method: "POST",
      body: JSON.stringify(body),
      headers,
    },
  ).then(res => res.json());

  return transformRetrieveTaxDetails(response);
}

function transformRetrieveTaxDetails(res: IResponseTaxEvidenceRequests): ITaxDetails | null {
  const taxDetails = res.tax_details && res.tax_details[0];

  if (!taxDetails) {
    return {
      id: res.id
    };
  }

  return {
    id: res.id,
    tax: res.tax,
    name: taxDetails.tax_name,
    rate: taxDetails.rate,
    jurisdiction: taxDetails.name,
  };
}
